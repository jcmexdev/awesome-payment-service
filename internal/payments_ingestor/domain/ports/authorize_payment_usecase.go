package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
)

type AuthorizePaymentUseCase interface {
	Execute(ctx context.Context, input *domain.AuthorizePaymentRequest) (*domain.Payment, error)
}
