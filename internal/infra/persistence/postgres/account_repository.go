package postgres

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/errors"
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
		return errors.NewAppError(errors.TypeInternal, "DATABASE_ERROR", "failed to create account", err).
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
