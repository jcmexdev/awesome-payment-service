package cache

import (
	"context"
	"log/slog"
	"time"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/ports"
)

type PersistenceCache struct {
	redis   ports.IdempotencyRepository
	memory  ports.IdempotencyRepository
	timeout time.Duration
}

func NewPersistenceCache(redis, memory ports.IdempotencyRepository, timeout time.Duration) *PersistenceCache {
	return &PersistenceCache{redis: redis, memory: memory, timeout: timeout}
}

func (p PersistenceCache) Lock(ctx context.Context, key string, ttl time.Duration) (bool, *domain.IdempotencyRecord, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	acquired, record, err := p.redis.Lock(ctxTimeout, key, ttl)
	if err == nil {
		return acquired, record, nil
	}

	slog.Warn("idempotency_primary_lock_failed_falling_back",
		slog.String("key", key),
		slog.String("error", err.Error()),
		slog.String("action", "using_local_fallback"),
	)

	acquiredFallback, recordFallback, errFallback := p.memory.Lock(ctx, key, ttl)
	if errFallback != nil {
		slog.Error("idempotency_fallback_lock_failed",
			slog.String("key", key),
			slog.String("error", errFallback.Error()),
		)
		return false, nil, errFallback
	}

	return acquiredFallback, recordFallback, nil
}

func (p PersistenceCache) Save(ctx context.Context, key string, statusCode int, body []byte, ttl time.Duration) error {
	if err := p.memory.Save(ctx, key, statusCode, body, ttl); err != nil {
		slog.Error("idempotency_fallback_save_failed", "key", key, "error", err)
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	if err := p.redis.Save(ctxTimeout, key, statusCode, body, ttl); err != nil {
		slog.Warn("idempotency_primary_save_failed",
			"key", key,
			"error", err,
			"action", "saved_to_local_only",
		)
	}

	return nil
}
