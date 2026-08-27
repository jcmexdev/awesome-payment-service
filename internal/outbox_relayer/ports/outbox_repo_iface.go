// internal/outbox_relayer/ports/outbox_repository.go
package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/outbox_relayer/domain"
)

type OutboxRepository interface {
	FetchPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkAsProcessed(ctx context.Context, eventID string) error
	MarkAsFailed(ctx context.Context, eventID string) error
}
