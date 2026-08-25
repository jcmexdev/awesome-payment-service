package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/domain"
)

type AuthorizePaymentUseCase interface {
	Execute(ctx context.Context, input *domain.AuthorizePaymentRequest) (*domain.Payment, error)
}
