package postgres

import (
	"context"
	"time"

	"github.com/jcmexdev/payment-service/pkg/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// FetchPendingEvents: Obtiene eventos en PENDING bloqueando filas concurrentes
func (r *OutboxRepository) FetchPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	var events []domain.OutboxEvent

	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "SKIP LOCKED",
		}).
		Where("status = ?", string(domain.OutboxStatusPending)).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error

	return events, err
}

func (r *OutboxRepository) FetchAndLockPendingEvents(
	ctx context.Context,
	batchSize int,
	baseIntervalSeconds int,
	maxAttempts int,
) ([]domain.OutboxEvent, error) { // 👈 Retornas directamente []domain.OutboxEvent
	var events []domain.OutboxEvent
	now := time.Now().UTC()

	query := `
		UPDATE outbox_events
		SET status = ?,
		    locked_until = NOW() + (? * INTERVAL '1 second' * power(2, attempts)),
		    attempts = attempts + 1
		WHERE id IN (
			SELECT id FROM outbox_events
			WHERE (status = ? OR (status = ? AND locked_until <= ?))
			  AND attempts < ?
			ORDER BY created_at ASC
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *;
	`

	// GORM escanea la respuesta de RETURNING directamente en la struct de dominio
	err := r.db.WithContext(ctx).Raw(
		query,
		string(domain.OutboxStatusProcessing),
		baseIntervalSeconds,
		string(domain.OutboxStatusPending),
		string(domain.OutboxStatusProcessing),
		now,
		maxAttempts,
		batchSize,
	).Scan(&events).Error

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (r *OutboxRepository) MarkAsProcessed(ctx context.Context, eventID string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&domain.OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"status":       string(domain.OutboxStatusProcessed),
			"processed_at": &now,
		}).Error
}

func (r *OutboxRepository) MarkAsFailed(ctx context.Context, eventID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.OutboxEvent{}).
		Where("id = ?", eventID).
		Update("status", string(domain.OutboxStatusFailed)).Error
}
