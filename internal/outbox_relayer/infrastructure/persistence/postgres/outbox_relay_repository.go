package postgres

import (
	"context"
	"time"

	"github.com/jcmexdev/payment-service/internal/outbox_relayer/domain"
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

// MarkAsProcessed: Marca el evento como PROCESSED tras publicarlo exitosamente
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

// MarkAsFailed: Si ocurre un error irrecuperable
func (r *OutboxRepository) MarkAsFailed(ctx context.Context, eventID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.OutboxEvent{}).
		Where("id = ?", eventID).
		Update("status", string(domain.OutboxStatusFailed)).Error
}
