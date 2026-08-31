package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/pkg/domain"
)

type PaymentsRepository interface {
	CreatePayment(ctx context.Context, payment *domain.Payment) error
}
