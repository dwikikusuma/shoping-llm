package logger

import (
	"log/slog"
	"os"
	"strings"
)

type Options struct {
	Service   string
	Env       string
	Level     string
	AddSource bool
}

func New(opts Options) *slog.Logger {
	level := parseLevel(opts.Level)

	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: opts.AddSource,
	})

	base := slog.New(h).With(
		"service", opts.Service,
		"env", opts.Env,
	)

	slog.SetDefault(base)
	return base
}

func parseLevel(lvl string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(lvl)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
