package ports

import (
	"context"
)

type PaymentUseCase interface {
	ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error
}
