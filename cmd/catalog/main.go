package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	cartv1 "github.com/dwikikusuma/shoping-llm/api/gen/cart/v1"
	catalogv1 "github.com/dwikikusuma/shoping-llm/api/gen/catalog/v1"
	checkoutv1 "github.com/dwikikusuma/shoping-llm/api/gen/checkout/v1"

	cartapp "github.com/dwikikusuma/shoping-llm/internal/cart/app"
	cartgrpc "github.com/dwikikusuma/shoping-llm/internal/cart/grpc"
	cartpg "github.com/dwikikusuma/shoping-llm/internal/cart/infra/postgres"

	catalogapp "github.com/dwikikusuma/shoping-llm/internal/catalog/app"
	cgrpc "github.com/dwikikusuma/shoping-llm/internal/catalog/grpc"
	cpg "github.com/dwikikusuma/shoping-llm/internal/catalog/infra/postgres"

	checkoutapp "github.com/dwikikusuma/shoping-llm/internal/checkout/app"
	checkoutgrpc "github.com/dwikikusuma/shoping-llm/internal/checkout/grpc"
	checkoutadapter "github.com/dwikikusuma/shoping-llm/internal/checkout/infra/adapter"

	"github.com/dwikikusuma/shoping-llm/pkg/config"
	"github.com/dwikikusuma/shoping-llm/pkg/logger"
	"github.com/dwikikusuma/shoping-llm/pkg/postgres"
	"github.com/dwikikusuma/shoping-llm/pkg/shutdown"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	log := logger.New(logger.Options{Service: "api", Env: cfg.AppEnv, Level: cfg.LogLevel, AddSource: true})

	ctx, cancel := shutdown.WithSignals(context.Background())
	defer cancel()

	db := mustDB(log)
	defer db.Close()

	// Catalog
	catalogRepo := cpg.NewProductRepo(db)
	catalogSvc := catalogapp.NewService(catalogRepo)

	// Cart
	cartRepo := cartpg.NewCartRepo(db)
	cartSvc := cartapp.NewService(cartRepo)

	// Checkout (adapters)
	cartReader := checkoutadapter.NewCartServiceReader(cartSvc)
	catalogReader := checkoutadapter.NewCatalogServiceReader(catalogSvc)
	checkoutSvc := checkoutapp.NewService(cartReader, catalogReader, 10)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("listen failed", slog.Any("err", err), slog.String("addr", addr))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(grpcServer, cgrpc.NewServer(catalogSvc))
	cartv1.RegisterCartServiceServer(grpcServer, cartgrpc.NewServer(cartSvc))
	checkoutv1.RegisterCheckoutServiceServer(grpcServer, checkoutgrpc.NewServer(checkoutSvc))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("grpc starting", slog.String("addr", addr))
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("grpc serve error", slog.Any("err", err))
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info("shutdown requested")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopCtx.Done():
		log.Warn("graceful stop timeout, forcing stop")
		grpcServer.Stop()
	case <-stopped:
	}

	wg.Wait()
	log.Info("bye")
}

func mustDB(log *slog.Logger) *sql.DB {
	cfg := postgres.Config{
		Host: getenv("POSTGRES_HOST", "localhost"),
		Port: getenvInt("POSTGRES_PORT", 5432),
		User: getenv("POSTGRES_USER", "shopping"),
		Pass: getenv("POSTGRES_PASSWORD", "shoppingpassword"),
		DB:   getenv("POSTGRES_DB", "shopping_db"),
	}
	db, err := postgres.Open(cfg)
	if err != nil {
		log.Error("db open failed", slog.Any("err", err))
		os.Exit(1)
	}
	return db
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
