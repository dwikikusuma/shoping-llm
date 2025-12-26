package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/dwikikusuma/shoping-llm/pkg/config"
	"github.com/dwikikusuma/shoping-llm/pkg/logger"
	"github.com/dwikikusuma/shoping-llm/pkg/shutdown"
)

func main() {
	cfg := config.Load()
	log := logger.New(logger.Options{
		Service:   "gateway",
		Env:       cfg.AppEnv,
		Level:     cfg.LogLevel,
		AddSource: true,
	})

	root := context.Background()
	ctx, cancel := shutdown.WithSignals(root)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("http server starting", slog.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", slog.Any("err", err))
			cancel()
		}
	}()

	<-ctx.Done()
	log.Info("shutdown requested")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown error", slog.Any("err", err))
	}

	wg.Wait()
	log.Info("bye")
}
