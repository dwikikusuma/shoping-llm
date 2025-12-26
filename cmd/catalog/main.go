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

	catalogv1 "github.com/dwikikusuma/shoping-llm/api/gen/catalog/v1"
	"github.com/dwikikusuma/shoping-llm/internal/catalog/app"
	cgrpc "github.com/dwikikusuma/shoping-llm/internal/catalog/grpc"
	cpg "github.com/dwikikusuma/shoping-llm/internal/catalog/infra/postgres"
	"github.com/dwikikusuma/shoping-llm/pkg/config"
	"github.com/dwikikusuma/shoping-llm/pkg/logger"
	"github.com/dwikikusuma/shoping-llm/pkg/postgres"
	"github.com/dwikikusuma/shoping-llm/pkg/shutdown"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	log := logger.New(logger.Options{Service: "catalog", Env: cfg.AppEnv, Level: cfg.LogLevel, AddSource: true})

	ctx, cancel := shutdown.WithSignals(context.Background())
	defer cancel()

	db := mustDB(log)
	defer db.Close()

	repo := cpg.NewProductRepo(db)
	svc := app.NewService(repo)

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("listen failed", slog.Any("err", err), slog.String("addr", addr))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	catalogv1.RegisterCatalogServiceServer(grpcServer, cgrpc.NewServer(svc))

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
