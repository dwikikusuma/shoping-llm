package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv   string
	LogLevel string

	GRPCPort int
	HTTPPort int
}

func Load() Config {
	return Config{
		AppEnv:   getEnv("APP_ENV", "dev"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		HTTPPort: getEnvInt("HTTP_PORT", 8080),
		GRPCPort: getEnvInt("GRPC_PORT", 8081),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
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
