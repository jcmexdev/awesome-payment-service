package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/pkg/domain/payment"
)

type OutboxRepository interface {
	Create(ctx context.Context, event *payment.OutboxEvent) error
}
