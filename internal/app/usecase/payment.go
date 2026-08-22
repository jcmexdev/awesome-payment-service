package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/constants"
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

func (u *PaymentUseCase) CreateAccount(ctx context.Context, account *domain.Account) error {
	tr := otel.Tracer("payment-service")
	ctx, span := tr.Start(ctx, "payment.create_account")
	defer span.End()

	err := u.ledgerRepo.CreateAccount(ctx, account)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "account created successfully")
	return nil
}

func (u *PaymentUseCase) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	tr := otel.Tracer("payment-service")
	ctx, span := tr.Start(ctx, "payment.get_account")
	defer span.End()

	account, err := u.ledgerRepo.GetAccount(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "account retrieved successfully")
	return account, nil
}

func (u *PaymentUseCase) ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
	tr := otel.Tracer("payment-service")
	ctx, span := tr.Start(ctx, "payment.process_payment")
	defer span.End()

	// Chaos Engineering: Inyección de Latencia
	if delay, ok := ctx.Value(constants.ContextKeySimulateDelay).(time.Duration); ok && delay > 0 {
		time.Sleep(delay)
	}

	// Chaos Engineering: Inyección de Errores
	if fail, ok := ctx.Value(constants.ContextKeySimulateError).(bool); ok && fail {
		err := errors.New("simulated transaction failure: database deadlock detected")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	err := u.ledgerRepo.ProcessPayment(ctx, accountID, amountCents, referenceID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "payment processed successfully")
	return nil
}
