package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
)

type CreateAccountUseCase interface {
	Execute(ctx context.Context, input *domain.CreateAccountRequest) (*domain.Account, error)
}
