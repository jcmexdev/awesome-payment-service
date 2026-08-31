package postgres

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
	"github.com/jcmexdev/payment-service/pkg/domain"
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
		return errors.NewAppError(constants.CodeDataBaseError,
			constants.CodeDataBaseError,
			constants.GetMessage(constants.CodeDataBaseError),
			err)

	}
	return nil
}
