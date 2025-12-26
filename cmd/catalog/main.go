package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/dwikikusuma/shoping-llm/pkg/config"
	"github.com/dwikikusuma/shoping-llm/pkg/logger"
	"github.com/dwikikusuma/shoping-llm/pkg/shutdown"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.Load()
	log := logger.New(logger.Options{
		Service:   "catalog",
		Env:       cfg.AppEnv,
		Level:     cfg.LogLevel,
		AddSource: true,
	})

	root := context.Background()
	ctx, cancel := shutdown.WithSignals(root)
	defer cancel()

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("failed to listen", slog.Any("err", err), slog.String("addr", addr))
		return
	}

	grpcServer := grpc.NewServer()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("grpc server starting", slog.String("addr", addr))
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
		log.Warn("grpc graceful stop timed out, forcing stop")
		grpcServer.Stop()
	case <-stopped:
	}

	wg.Wait()
	log.Info("bye")
}
