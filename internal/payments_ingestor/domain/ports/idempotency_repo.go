package ports

import (
	"context"
	"time"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
)

type IdempotencyRepository interface {
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, *domain.IdempotencyRecord, error)
	Save(ctx context.Context, key string, statusCode int, body []byte, ttl time.Duration) error
}
