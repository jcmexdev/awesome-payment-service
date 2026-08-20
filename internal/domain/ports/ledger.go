package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/domain"
)

type LedgerRepository interface {
	CreateAccount(ctx context.Context, account *domain.Account) error
	GetAccount(ctx context.Context, id string) (*domain.Account, error)
	ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error
}
