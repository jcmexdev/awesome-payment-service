package usecase

import (
	"context"

	"github.com/jcmexdev/payment-service/internal/domain/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

type PaymentUseCase struct {
	ledgerRepo ports.LedgerRepository
}

func NewPaymentUseCase(ledgerRepo ports.LedgerRepository) *PaymentUseCase {
	return &PaymentUseCase{ledgerRepo: ledgerRepo}
}

func (u *PaymentUseCase) ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
	tr := otel.Tracer("payment-service")
	ctx, span := tr.Start(ctx, "payment.process_payment")
	defer span.End()

	err := u.ledgerRepo.ProcessPayment(ctx, accountID, amountCents, referenceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "payment processed successfully")
	return nil
}
