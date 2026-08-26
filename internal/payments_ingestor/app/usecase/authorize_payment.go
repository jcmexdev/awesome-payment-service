package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	domain2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	errors2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
	ports2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/ports"
)

type AuthorizePaymentUseCase struct {
	paymentsRepo ports2.PaymentsRepository
	outboxRepo   ports2.OutboxRepository
	uow          ports2.UnitOfWork
}

func NewAuthorizePaymentUseCase(paymentsRepo ports2.PaymentsRepository, outboxRepo ports2.OutboxRepository, uow ports2.UnitOfWork) *AuthorizePaymentUseCase {
	return &AuthorizePaymentUseCase{paymentsRepo: paymentsRepo, outboxRepo: outboxRepo, uow: uow}
}

func (a AuthorizePaymentUseCase) Execute(ctx context.Context, input *domain2.AuthorizePaymentRequest) (*domain2.Payment, error) {
	payment := &domain2.Payment{
		ID:        uuid.NewString(),
		AccountID: input.AccountID,
		Amount:    input.Amount,
		Currency:  input.Currency,
		Status:    string(constants.PaymentCreated),
		CreatedAt: time.Now(),
	}

	payloadBytes, err := json.Marshal(payment)
	if err != nil {
		return nil, errors2.NewAppError(
			constants.TypeInternal,
			constants.CodeMalformedJSON,
			constants.GetMessage(constants.CodeMalformedJSON),
			err,
		)
	}

	outboxEvent := &domain2.OutboxEvent{
		ID:            uuid.NewString(),
		AggregateType: "PAYMENT",
		AggregateID:   payment.ID,
		EventType:     "PAYMENT_CREATED",
		Payload:       payloadBytes,
		Status:        "PENDING",
		CreatedAt:     payment.CreatedAt,
	}

	err = a.uow.Do(ctx, func(ctx context.Context) error {
		err := a.paymentsRepo.CreatePayment(ctx, payment)
		if err != nil {
			return err
		}

		err = a.outboxRepo.Create(ctx, outboxEvent)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return payment, nil
}

/*func NewPaymentUseCase(ledgerRepo ports.LedgerRepository) *AuthorizePaymentUseCase {
	return &AuthorizePaymentUseCase{ledgerRepo: ledgerRepo}
}

func (u *AuthorizePaymentUseCase) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
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
	return nil, nil
}

func (u *AuthorizePaymentUseCase) ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
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
}*/
