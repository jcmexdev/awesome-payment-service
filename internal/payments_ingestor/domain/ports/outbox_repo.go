package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
)

type OutboxRepository interface {
	Create(ctx context.Context, event *domain.OutboxEvent) error
}
