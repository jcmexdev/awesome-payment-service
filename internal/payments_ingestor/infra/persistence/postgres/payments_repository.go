package postgres

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	errors2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
	"gorm.io/gorm"
)

type PaymentsRepository struct {
	gormDB *gorm.DB
}

func NewPaymentsRepository(gormDB *gorm.DB) *PaymentsRepository {
	return &PaymentsRepository{gormDB: gormDB}
}

func (p PaymentsRepository) CreatePayment(ctx context.Context, payment *domain.Payment) error {
	db := getTx(ctx, p.gormDB)
	err := db.Create(payment).Error
	if err != nil {
		return errors2.NewAppError(constants.CodeDataBaseError,
			constants.CodeDataBaseError,
			constants.GetMessage(constants.CodeDataBaseError),
			err)

	}
	return nil
}
