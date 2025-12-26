package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func WithSignals(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer signal.Stop(ch)
		select {
		case <-ctx.Done():
			return
		case <-ch:
			cancel()
		}
	}()

	return ctx, cancel
}
