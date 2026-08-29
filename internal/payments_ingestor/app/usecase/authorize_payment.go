package usecase

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/ports"
	"github.com/jcmexdev/payment-service/pkg/domain/payment"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

type AuthorizePaymentUseCase struct {
	paymentsRepo ports.PaymentsRepository
	outboxRepo   ports.OutboxRepository
	uow          ports.UnitOfWork
}

func NewAuthorizePaymentUseCase(paymentsRepo ports.PaymentsRepository, outboxRepo ports.OutboxRepository, uow ports.UnitOfWork) *AuthorizePaymentUseCase {
	return &AuthorizePaymentUseCase{paymentsRepo: paymentsRepo, outboxRepo: outboxRepo, uow: uow}
}

func (a AuthorizePaymentUseCase) Execute(ctx context.Context, in *domain.AuthorizePaymentRequest) (*payment.Payment, error) {
	tr := otel.Tracer("payments_ingestor")
	ctx, span := tr.Start(ctx, "payment.authorize_payment")
	defer span.End()

	newPayment, err := payment.NewPayment(uuid.NewString(), in.Amount, in.Currency, in.AccountID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	payloadBytes, err := json.Marshal(newPayment)
	if err != nil {
		appErr := errors.NewAppError(
			constants.TypeInternal,
			constants.CodeMalformedJSON,
			constants.GetMessage(constants.CodeMalformedJSON),
			err,
		)
		span.RecordError(appErr)
		span.SetStatus(codes.Error, appErr.Error())
		return nil, appErr
	}

	// Inyectar contexto de trazabilidad W3C (traceparent, tracestate, baggage)
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	traceContextBytes, err := json.Marshal(carrier)
	if err != nil {
		traceContextBytes = []byte("{}")
	}

	outboxEvent := payment.NewOutboxEvent(
		uuid.NewString(),
		"PAYMENT",
		newPayment.ID,
		"payment.created",
		payloadBytes,
		traceContextBytes,
		payment.OutboxStatusPending,
	)

	err = a.uow.Do(ctx, func(ctx context.Context) error {
		err := a.paymentsRepo.CreatePayment(ctx, newPayment)
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "payment authorized successfully")
	return newPayment, nil
}
