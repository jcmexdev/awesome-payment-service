package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
)

type PaymentsRepository interface {
	CreatePayment(ctx context.Context, payment *domain.Payment) error
}
