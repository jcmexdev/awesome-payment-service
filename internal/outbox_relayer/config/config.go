package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment       string
	DBurl             string
	PollInterval      time.Duration
	BatchSize         int
	OtelCollectorAddr string
	OtelServiceName   string
}

// LoadConfig lee exclusivamente las variables de entorno inyectadas en el SO (por Docker Compose)
func LoadConfig() *Config {
	return &Config{
		Environment:       getEnv("APP_ENV", "development"),
		DBurl:             getEnv("DATABASE_URL", "localhost"),
		PollInterval:      time.Duration(getEnvAsInt("POLL_INTERVAL_MS", 1000)) * time.Millisecond,
		BatchSize:         getEnvAsInt("BATCH_SIZE", 100),
		OtelCollectorAddr: getEnv("OTEL_COLLECTOR_ADDR", ""),
		OtelServiceName:   getEnv("OTEL_SERVICE_NAME", "outbox_relayer"),
	}
}

// Helper para obtener string con valor por defecto
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// Helper para convertir tipos int de forma segura
func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}
