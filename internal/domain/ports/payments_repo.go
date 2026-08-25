package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/domain"
)

type PaymentsRepository interface {
	CreatePayment(ctx context.Context, payment *domain.Payment) error
}
