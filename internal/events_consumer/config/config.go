package config

import (
	"os"
	"strconv"
)

type Config struct {
	Environment       string
	DBurl             string
	OtelCollectorAddr string
	OtelServiceName   string
	SQSEndpoint       string
	SQSRegion         string
	SQSUrl            string
}

// LoadConfig lee exclusivamente las variables de entorno inyectadas en el SO (por Docker Compose)
func LoadConfig() *Config {
	return &Config{
		Environment:       getEnv("APP_ENV", "development"),
		DBurl:             getEnv("DATABASE_URL", "localhost"),
		OtelCollectorAddr: getEnv("OTEL_COLLECTOR_ADDR", ""),
		SQSEndpoint:       getEnv("SQS_ENDPOINT", "outbox_relayer"),
		SQSRegion:         getEnv("SQS_REGION", "outbox_relayer"),
		SQSUrl:            getEnv("SQS_QUEUE_URL", "outbox_relayer"),
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
