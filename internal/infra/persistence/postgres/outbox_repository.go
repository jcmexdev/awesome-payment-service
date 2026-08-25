package postgres

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/errors"
	"gorm.io/gorm"
)

type OutboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (o OutboxRepository) Create(ctx context.Context, event *domain.OutboxEvent) error {
	db := getTx(ctx, o.db)
	err := db.Create(event).Error
	if err != nil {
		return errors.NewAppError(errors.TypeInternal,
			errors.CodeDataBaseError,
			errors.GetMessage(errors.CodeDataBaseError),
			err,
		)
	}
	return nil
}
