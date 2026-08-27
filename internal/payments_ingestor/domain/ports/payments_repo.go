package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/pkg/domain/payment"
)

type PaymentsRepository interface {
	CreatePayment(ctx context.Context, payment *payment.Payment) error
}
