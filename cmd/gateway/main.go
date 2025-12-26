package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	catalogv1 "github.com/dwikikusuma/shoping-llm/api/gen/catalog/v1"
	"github.com/dwikikusuma/shoping-llm/pkg/config"
	"github.com/dwikikusuma/shoping-llm/pkg/logger"
	"github.com/dwikikusuma/shoping-llm/pkg/shutdown"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type server struct {
	log     *slog.Logger
	catalog catalogv1.CatalogServiceClient
}

func main() {
	cfg := config.Load()
	log := logger.New(logger.Options{Service: "gateway", Env: cfg.AppEnv, Level: cfg.LogLevel, AddSource: true})

	ctx, cancel := shutdown.WithSignals(context.Background())
	defer cancel()

	//conn, err := grpc.Dial(cfg.CatalogGRPCAddr, grpc.WithInsecure())
	conn, err := grpc.NewClient(cfg.CatalogGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("grpc dial failed", slog.Any("err", err), slog.String("addr", cfg.CatalogGRPCAddr))
		return
	}
	defer conn.Close()

	s := &server{
		log:     log,
		catalog: catalogv1.NewCatalogServiceClient(conn),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	mux.HandleFunc("/v1/products", s.productsHandler)
	mux.HandleFunc("/v1/products/", s.productByIDHandler)

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           withReqID(log, mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("http starting", slog.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", slog.Any("err", err))
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info("shutdown requested")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)

	wg.Wait()
	log.Info("bye")
}

func withReqID(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = newReqID()
		}
		w.Header().Set("X-Request-Id", rid)
		ctx := context.WithValue(r.Context(), reqIDKey{}, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type reqIDKey struct{}

func reqIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(reqIDKey{}).(string)
	return v
}

func newReqID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type createProductReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       struct {
		Currency string `json:"currency"`
		Amount   int64  `json:"amount"`
	} `json:"price"`
}

type productResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       struct {
		Currency string `json:"currency"`
		Amount   int64  `json:"amount"`
	} `json:"price"`
	CreatedAtUnix int64 `json:"created_at_unix"`
	UpdatedAtUnix int64 `json:"updated_at_unix"`
}

type listProductsResp struct {
	Products   []productResp `json:"products"`
	NextCursor string        `json:"next_cursor"`
}

func (s *server) productsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createProductHTTP(w, r)
	case http.MethodGet:
		s.listProductsHTTP(w, r)
	default:
		writeErr(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) productByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/products/")
	id = strings.TrimSpace(id)
	if id == "" {
		writeErr(w, "missing id", http.StatusBadRequest)
		return
	}
	s.getProductHTTP(w, r, id)
}

func (s *server) createProductHTTP(w http.ResponseWriter, r *http.Request) {
	var body createProductReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, "invalid json", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.catalog.CreateProduct(ctx, &catalogv1.CreateProductRequest{
		Name:        body.Name,
		Description: body.Description,
		Price: &catalogv1.Money{
			Currency: body.Price.Currency,
			Amount:   body.Price.Amount,
		},
	})
	if err != nil {
		s.log.Error("create product failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())))
		writeErr(w, "create failed", http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, toHTTPProduct(resp.Product))
}

func (s *server) getProductHTTP(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.catalog.GetProduct(ctx, &catalogv1.GetProductRequest{Id: id})
	if err != nil {
		s.log.Error("get product failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())), slog.String("id", id))
		writeErr(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, toHTTPProduct(resp.Product))
}

func (s *server) listProductsHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("query")
	cursor := r.URL.Query().Get("cursor")

	limit := 20
	if v := r.URL.Query().Get("limit"); strings.TrimSpace(v) != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.catalog.ListProducts(ctx, &catalogv1.ListProductsRequest{
		Query:  q,
		Limit:  int32(limit),
		Cursor: cursor,
	})
	if err != nil {
		s.log.Error("list products failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())))
		writeErr(w, err.Error(), http.StatusBadRequest)
		return
	}

	out := listProductsResp{
		Products:   make([]productResp, 0, len(resp.Products)),
		NextCursor: resp.NextCursor,
	}
	for _, p := range resp.Products {
		out.Products = append(out.Products, toHTTPProduct(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func toHTTPProduct(p *catalogv1.Product) productResp {
	var out productResp
	out.ID = p.GetId()
	out.Name = p.GetName()
	out.Description = p.GetDescription()
	out.Price.Currency = p.GetPrice().GetCurrency()
	out.Price.Amount = p.GetPrice().GetAmount()
	out.CreatedAtUnix = p.GetCreatedAtUnix()
	out.UpdatedAtUnix = p.GetUpdatedAtUnix()
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errResp struct {
	Error string `json:"error"`
}

func writeErr(w http.ResponseWriter, msg string, status int) {
	writeJSON(w, status, errResp{Error: msg})
}
