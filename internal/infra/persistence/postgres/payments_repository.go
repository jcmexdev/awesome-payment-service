package postgres

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/errors"
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
		return errors.NewAppError(errors.CodeDataBaseError,
			errors.CodeDataBaseError,
			errors.GetMessage(errors.CodeDataBaseError),
			err)

	}
	return nil
}
