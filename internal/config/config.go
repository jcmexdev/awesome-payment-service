package config

import (
	"os"
	"time"
)

type Config struct {
	Port                string
	RedisAddr           string
	RedisTimeout        time.Duration
	LogLevel            string
	IdempotencyFilePath string
	IdempotencyTTL      time.Duration
	OtelCollectorAddr   string
	OtelServiceName     string
	DatabaseURL         string
	ServiceName         string
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		RedisTimeout:        parseDuration(getEnv("REDIS_TIMEOUT", "50ms"), 50*time.Millisecond),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		IdempotencyFilePath: getEnv("IDEMPOTENCY_FILE_PATH", "idempotency.db"),
		IdempotencyTTL:      parseDuration(getEnv("IDEMPOTENCY_TTL", "1h"), 1*time.Hour),
		OtelCollectorAddr:   getEnv("OTEL_COLLECTOR_ADDR", ""),
		OtelServiceName:     getEnv("OTEL_SERVICE_NAME", "payment-service"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"),
		ServiceName:         getEnv("SERVICE_NAME", "payment-service"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(value string, defaultValue time.Duration) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return d
}
