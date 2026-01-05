package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	cartv1 "github.com/dwikikusuma/shoping-llm/api/gen/cart/v1"
	catalogv1 "github.com/dwikikusuma/shoping-llm/api/gen/catalog/v1"
	checkoutv1 "github.com/dwikikusuma/shoping-llm/api/gen/checkout/v1"

	"github.com/dwikikusuma/shoping-llm/pkg/config"
	"github.com/dwikikusuma/shoping-llm/pkg/logger"
	"github.com/dwikikusuma/shoping-llm/pkg/shutdown"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type server struct {
	log      *slog.Logger
	catalog  catalogv1.CatalogServiceClient
	cart     cartv1.CartServiceClient
	checkout checkoutv1.CheckoutServiceClient
}

func main() {
	cfg := config.Load()
	log := logger.New(logger.Options{Service: "gateway", Env: cfg.AppEnv, Level: cfg.LogLevel, AddSource: true})

	ctx, cancel := shutdown.WithSignals(context.Background())
	defer cancel()

	conn, err := grpc.NewClient(cfg.CatalogGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("grpc dial failed", slog.Any("err", err), slog.String("addr", cfg.CatalogGRPCAddr))
		return
	}
	defer conn.Close()

	s := &server{
		log:      log,
		catalog:  catalogv1.NewCatalogServiceClient(conn),
		cart:     cartv1.NewCartServiceClient(conn),
		checkout: checkoutv1.NewCheckoutServiceClient(conn),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	// Catalog
	mux.HandleFunc("/v1/products", s.productsHandler)
	mux.HandleFunc("/v1/products/", s.productByIDHandler)

	// Cart + Checkout
	mux.HandleFunc("/v1/cart/", s.cartHandler)
	mux.HandleFunc("/v1/checkout/quote/", s.quoteHandler)

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

/* =========================
   Catalog HTTP (existing)
   ========================= */

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
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
		return // IMPORTANT
	}

	writeJSON(w, http.StatusCreated, toHTTPProduct(resp.Product))
}

func (s *server) getProductHTTP(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.catalog.GetProduct(ctx, &catalogv1.GetProductRequest{Id: id})
	if err != nil {
		s.log.Error("get product failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())), slog.String("id", id))
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toHTTPProduct(resp.Product))
}

func (s *server) listProductsHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("query")
	cursor := r.URL.Query().Get("cursor")

	limit := 20
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
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
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
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

/* =========================
   Cart HTTP
   ========================= */

type cartItemHTTP struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type cartHTTP struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Status    string         `json:"status"`
	Items     []cartItemHTTP `json:"items"`
	CreatedAt int64          `json:"created_at_unix"`
	UpdatedAt int64          `json:"updated_at_unix"`
}

type addItemReq struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type setQtyReq struct {
	Quantity int32 `json:"quantity"`
}

// Routes:
// GET    /v1/cart/{user_id}
// POST   /v1/cart/{user_id}/items
// PUT    /v1/cart/{user_id}/items/{product_id}
// DELETE /v1/cart/{user_id}/items/{product_id}
// DELETE /v1/cart/{user_id}/items
func (s *server) cartHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/cart/")
	path = strings.Trim(path, "/")
	parts := []string{}
	if path != "" {
		parts = strings.Split(path, "/")
	}

	// Need at least user_id
	if len(parts) < 1 || strings.TrimSpace(parts[0]) == "" {
		writeErr(w, "missing user_id", http.StatusBadRequest)
		return
	}
	userID := parts[0]

	// /v1/cart/{user_id}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeErr(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.getOrCreateCartHTTP(w, r, userID)
		return
	}

	// /v1/cart/{user_id}/items
	if len(parts) == 2 && parts[1] == "items" {
		switch r.Method {
		case http.MethodPost:
			s.addItemHTTP(w, r, userID)
		case http.MethodDelete:
			s.clearCartHTTP(w, r, userID)
		default:
			writeErr(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /v1/cart/{user_id}/items/{product_id}
	if len(parts) == 3 && parts[1] == "items" {
		productID := parts[2]
		switch r.Method {
		case http.MethodPut:
			s.setItemQtyHTTP(w, r, userID, productID)
		case http.MethodDelete:
			s.removeItemHTTP(w, r, userID, productID)
		default:
			writeErr(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	writeErr(w, "not found", http.StatusNotFound)
}

func (s *server) getOrCreateCartHTTP(w http.ResponseWriter, r *http.Request, userID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.cart.GetOrCreateCart(ctx, &cartv1.UserId{Id: userID})
	if err != nil {
		s.log.Error("get or create cart failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())), slog.String("user_id", userID))
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, toHTTPCart(resp))
}

func (s *server) addItemHTTP(w http.ResponseWriter, r *http.Request, userID string) {
	var body addItemReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, "invalid json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.ProductID) == "" {
		writeErr(w, "missing product_id", http.StatusBadRequest)
		return
	}
	if body.Quantity <= 0 {
		writeErr(w, "quantity must be > 0", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.cart.AddItem(ctx, &cartv1.UpdateCartItemRequest{
		UserId: userID,
		Item: &cartv1.CartItem{
			ProductId: body.ProductID,
			Quantity:  body.Quantity,
		},
	})
	if err != nil {
		s.log.Error("add item failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())), slog.String("user_id", userID))
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, toHTTPCart(resp))
}

func (s *server) setItemQtyHTTP(w http.ResponseWriter, r *http.Request, userID, productID string) {
	var body setQtyReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Quantity <= 0 {
		writeErr(w, "quantity must be > 0", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.cart.SetItemQuantity(ctx, &cartv1.UpdateCartItemRequest{
		UserId: userID,
		Item: &cartv1.CartItem{
			ProductId: productID,
			Quantity:  body.Quantity,
		},
	})
	if err != nil {
		s.log.Error("set item quantity failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())), slog.String("user_id", userID))
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, toHTTPCart(resp))
}

func (s *server) removeItemHTTP(w http.ResponseWriter, r *http.Request, userID, productID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.cart.RemoveItem(ctx, &cartv1.RemoveCartItemRequest{
		UserId:    userID,
		ProductId: productID,
	})
	if err != nil {
		s.log.Error("remove item failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())), slog.String("user_id", userID))
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, toHTTPCart(resp))
}

func (s *server) clearCartHTTP(w http.ResponseWriter, r *http.Request, userID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// NOTE: your current Cart gRPC ClearCart expects CartId but uses it like a user_id in server code.
	resp, err := s.cart.ClearCart(ctx, &cartv1.CartId{Id: userID})
	if err != nil {
		s.log.Error("clear cart failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())), slog.String("user_id", userID))
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
		return
	}

	writeJSON(w, http.StatusOK, toHTTPCart(resp))
}

func toHTTPCart(c *cartv1.Cart) cartHTTP {
	out := cartHTTP{
		ID:        c.GetId(),
		UserID:    c.GetUserId(),
		Status:    c.GetStatus(),
		CreatedAt: c.GetCreatedAtUnix(),
		UpdatedAt: c.GetUpdatedAtUnix(),
		Items:     make([]cartItemHTTP, 0, len(c.GetItems())),
	}
	for _, it := range c.GetItems() {
		out.Items = append(out.Items, cartItemHTTP{
			ProductID: it.GetProductId(),
			Quantity:  it.GetQuantity(),
		})
	}
	return out
}

/* =========================
   Checkout Quote HTTP
   ========================= */

// GET /v1/checkout/quote/{user_id}
func (s *server) quoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := strings.TrimPrefix(r.URL.Path, "/v1/checkout/quote/")
	userID = strings.Trim(userID, "/")
	userID = strings.TrimSpace(userID)
	if userID == "" {
		writeErr(w, "missing user_id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp, err := s.checkout.Quote(ctx, &checkoutv1.QuoteRequest{UserId: userID})
	if err != nil {
		s.log.Error("quote failed", slog.Any("err", err), slog.String("rid", reqIDFrom(r.Context())), slog.String("user_id", userID))
		httpCode, code, msg := httpStatusFromGRPC(err)
		writeAPIError(w, httpCode, code, msg)
		return
	}

	// resp already JSON-friendly, but we’ll map explicitly.
	writeJSON(w, http.StatusOK, resp)
}

/* =========================
   Common HTTP utils
   ========================= */

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

type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func writeAPIError(w http.ResponseWriter, statusCode int, code string, msg string) {
	writeJSON(w, statusCode, apiError{Error: msg, Code: code})
}

func httpStatusFromGRPC(err error) (int, string, string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, "INTERNAL", "internal error"
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return http.StatusBadRequest, "INVALID_ARGUMENT", st.Message()
	case codes.NotFound:
		return http.StatusNotFound, "NOT_FOUND", st.Message()
	case codes.Unavailable, codes.DeadlineExceeded:
		return http.StatusServiceUnavailable, "UNAVAILABLE", st.Message()
	default:
		_ = sql.ErrNoRows
		return http.StatusInternalServerError, "INTERNAL", "internal error"
	}
}
