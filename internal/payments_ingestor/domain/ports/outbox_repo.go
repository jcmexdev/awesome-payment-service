package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/pkg/domain"
)

type OutboxRepository interface {
	Create(ctx context.Context, event *domain.OutboxEvent) error
}
