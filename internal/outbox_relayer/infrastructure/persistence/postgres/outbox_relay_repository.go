package postgres

import (
	"context"
	"time"

	"github.com/jcmexdev/payment-service/pkg/domain/payment"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// Helper para extraer la transacción activa si viene dentro del contexto del UnitOfWork
func getTx(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value("tx_key").(*gorm.DB); ok {
		return tx
	}
	return defaultDB
}

// FetchPendingEvents: Obtiene eventos en PENDING bloqueando filas concurrentes
func (r *OutboxRepository) FetchPendingEvents(ctx context.Context, limit int) ([]payment.OutboxEvent, error) {
	var events []payment.OutboxEvent

	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "SKIP LOCKED",
		}).
		Where("status = ?", string(payment.OutboxStatusPending)).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error

	return events, err
}

func (r *OutboxRepository) MarkAsProcessed(ctx context.Context, eventID string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&payment.OutboxEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"status":       string(payment.OutboxStatusProcessed),
			"processed_at": &now,
		}).Error
}

func (r *OutboxRepository) MarkAsFailed(ctx context.Context, eventID string) error {
	return r.db.WithContext(ctx).
		Model(&payment.OutboxEvent{}).
		Where("id = ?", eventID).
		Update("status", string(payment.OutboxStatusFailed)).Error
}
