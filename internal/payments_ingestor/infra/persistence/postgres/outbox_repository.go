package postgres

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	errors "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
	"github.com/jcmexdev/payment-service/pkg/domain"
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
		return errors.NewAppError(constants.TypeInternal,
			constants.CodeDataBaseError,
			constants.GetMessage(constants.CodeDataBaseError),
			err,
		)
	}
	return nil
}
