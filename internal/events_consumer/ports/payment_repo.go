package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/pkg/domain"
)

type PaymentRepository interface {
	FindByID(ctx context.Context, ID string) (*domain.Payment, error)
	UpdateStatus(ctx context.Context, ID string, status domain.PaymentStatus) error
}
