package postgres

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	errors2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (a AccountRepository) CreateAccount(ctx context.Context, account *domain.Account) error {

	if err := a.db.WithContext(ctx).Create(account).Error; err != nil {
		return errors2.NewAppError(constants.TypeInternal, "DATABASE_ERROR", "failed to create account", err).
			WithLogContext("user_id", account.UserID).
			WithLogContext("currency", account.Currency)
	}
	return nil
}

func (a AccountRepository) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	//TODO implement me
	panic("implement me")
}

func (a AccountRepository) UpdateAccount(ctx context.Context, account *domain.Account) error {
	//TODO implement me
	panic("implement me")

}
