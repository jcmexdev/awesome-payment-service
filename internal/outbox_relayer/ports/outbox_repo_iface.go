// internal/outbox_relayer/ports/outbox_repository.go
package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/pkg/domain"
)

type OutboxRepository interface {
	FetchPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkAsProcessed(ctx context.Context, eventID string) error
	MarkAsFailed(ctx context.Context, eventID string) error
	FetchAndLockPendingEvents(
		ctx context.Context,
		batchSize int,
		baseIntervalSeconds int,
		maxAttempts int,
	) ([]domain.OutboxEvent, error)
}
