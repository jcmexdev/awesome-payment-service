package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/domain"
)

type CreateAccountUseCase interface {
	Execute(ctx context.Context, input *domain.CreateAccountRequest) (*domain.Account, error)
}
