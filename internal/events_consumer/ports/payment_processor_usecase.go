package ports

import "context"

type PaymentProcessorUseCase interface {
	ProcessTransaction(ctx context.Context, payload []byte) error
}
