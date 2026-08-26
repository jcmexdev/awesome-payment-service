package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
)

type AccountRepository interface {
	CreateAccount(ctx context.Context, account *domain.Account) error
	GetAccount(ctx context.Context, id string) (*domain.Account, error)
	UpdateAccount(ctx context.Context, account *domain.Account) error
}
