package ports

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/pkg/domain/payment"
)

type AuthorizePaymentUseCase interface {
	Execute(ctx context.Context, input *domain.AuthorizePaymentRequest) (*payment.Payment, error)
}
