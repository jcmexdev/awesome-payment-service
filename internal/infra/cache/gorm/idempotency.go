package gorm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/infra/telemetry"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/opentelemetry/tracing"
)

type IdempotencyCache struct {
	db *gorm.DB
}

func NewConnection(ctx context.Context, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("database dsn required")
	}

	gormLogger := telemetry.NewGormSlogLogger(slog.Default(), 200*time.Millisecond)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	// 1. Registro del Plugin de OpenTelemetry para GORM
	if err := db.Use(tracing.NewPlugin(
		tracing.WithDBSystem("payments_db"),
	)); err != nil {
		return nil, fmt.Errorf("failed to register opentelemetry plugin: %w", err)
	}

	// 2. Configuración del Pool de Conexiones nativo (*sql.DB)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configuración optimizada para PostgreSQL en producción
	sqlDB.SetMaxOpenConns(25)                 // Límite de conexiones abiertas
	sqlDB.SetMaxIdleConns(10)                 // Conexiones inactivas en reserva
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Reutilización de sockets

	// 3. Health Check (Ping) con Timeout
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	// 4. Auto-migración con propagación de Contexto
	if err := db.WithContext(ctx).AutoMigrate(
		&domain.Account{},
		&domain.LedgerEntry{},
		&domain.IdempotencyRecord{},
	); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database auto-migration failed: %w", err)
	}

	return db, nil
}

func NewIdempotencyCache(ctx context.Context, db *gorm.DB) (*IdempotencyCache, error) {
	return &IdempotencyCache{db: db}, nil
}

func (i IdempotencyCache) Lock(ctx context.Context, key string, ttl time.Duration) (bool, *domain.IdempotencyRecord, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	// Clean up expired records
	_ = i.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&domain.IdempotencyRecord{}).Error

	rec := domain.IdempotencyRecord{
		Key:       key,
		Status:    domain.IdempotencyStatusProcessing,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Try to create the record (acting as lock acquisition)
	err := i.db.WithContext(ctx).Create(&rec).Error
	if err == nil {
		return true, nil, nil
	}

	// If creation failed, select the existing record to inspect its status
	var existing domain.IdempotencyRecord
	if err := i.db.WithContext(ctx).Where("key = ?", key).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to query existing idempotency record: %w", err)
	}

	return false, &existing, nil
}

func (i IdempotencyCache) Save(ctx context.Context, key string, statusCode int, body []byte, ttl time.Duration) error {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	rec := domain.IdempotencyRecord{
		Key:          key,
		Status:       domain.IdempotencyStatusCompleted,
		ResponseCode: statusCode,
		ResponseBody: body,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err := i.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "response_code", "response_body", "expires_at", "updated_at"}),
	}).Create(&rec).Error
	if err != nil {
		return fmt.Errorf("database upsert failed: %w", err)
	}

	return nil
}
