package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	pkgDomain "github.com/jcmexdev/payment-service/pkg/domain"
)

type AuthorizePaymentUseCase interface {
	Execute(ctx context.Context, input *domain.AuthorizePaymentRequest) (*pkgDomain.Payment, error)
}
