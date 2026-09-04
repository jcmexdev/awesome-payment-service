package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jcmexdev/payment-service/pkg/domain"
	"gorm.io/gorm"
)

type PaymentsRepository struct {
	db *gorm.DB
}

func NewPaymentsRepository(db *gorm.DB) *PaymentsRepository {
	return &PaymentsRepository{db: db}
}

func (p PaymentsRepository) FindByID(ctx context.Context, ID string) (*domain.Payment, error) {
	pay := &domain.Payment{}
	err := p.db.WithContext(ctx).First(pay, "id = ?", ID).Error
	if err != nil {
		return nil, err
	}
	return pay, nil
}

func (p PaymentsRepository) UpdateStatus(ctx context.Context, ID string, status domain.PaymentStatus) error {
	now := time.Now().UTC()

	result := p.db.WithContext(ctx).
		Model(&domain.Payment{}).
		Where("id = ?", ID).
		Updates(map[string]interface{}{
			"status":     string(status),
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update payment status for ID %s: %w", ID, result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("payment with ID %s not found", ID)
	}

	return nil
}
