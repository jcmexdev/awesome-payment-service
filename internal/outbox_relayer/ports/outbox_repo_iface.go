// internal/outbox_relayer/ports/outbox_repository.go
package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/pkg/domain/payment"
)

type OutboxRepository interface {
	FetchPendingEvents(ctx context.Context, limit int) ([]payment.OutboxEvent, error)
	MarkAsProcessed(ctx context.Context, eventID string) error
	MarkAsFailed(ctx context.Context, eventID string) error
}
