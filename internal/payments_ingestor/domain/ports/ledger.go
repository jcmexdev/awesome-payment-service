package ports

import (
	"context"
)

type LedgerRepository interface {
	ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error
}
