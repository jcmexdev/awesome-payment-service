package usecase_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/app/usecase"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/pkg/domain/payment"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type mockPaymentsRepo struct {
	createdPayment *payment.Payment
}

func (m *mockPaymentsRepo) CreatePayment(ctx context.Context, p *payment.Payment) error {
	m.createdPayment = p
	return nil
}

type mockOutboxRepo struct {
	createdEvent *payment.OutboxEvent
}

func (m *mockOutboxRepo) Create(ctx context.Context, event *payment.OutboxEvent) error {
	m.createdEvent = event
	return nil
}

type mockUoW struct {
	invoked bool
}

func (m *mockUoW) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	m.invoked = true
	return fn(ctx)
}

func TestAuthorizePaymentUseCase_PersistsPaymentAndOutboxWithTraceContext(t *testing.T) {
	// 1. Configurar TracerProvider y W3C Propagator para el test
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	paymentsRepo := &mockPaymentsRepo{}
	outboxRepo := &mockOutboxRepo{}
	uow := &mockUoW{}

	uc := usecase.NewAuthorizePaymentUseCase(paymentsRepo, outboxRepo, uow)

	// 2. Ejecutar caso de uso con contexto activo
	ctx := context.Background()
	req := &domain.AuthorizePaymentRequest{
		Amount:    15000,
		Currency:  "USD",
		AccountID: "acc-user-999",
	}

	result, err := uc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error executing AuthorizePayment: %v", err)
	}

	// 3. Validar resultado
	if result == nil {
		t.Fatal("expected non-nil payment result")
	}
	if result.Amount != 15000 || result.Currency != "USD" || result.AccountID != "acc-user-999" {
		t.Errorf("payment data mismatch: %+v", result)
	}

	// 4. Validar que la UnitOfWork fue ejecutada
	if !uow.invoked {
		t.Error("expected UnitOfWork.Do to be executed")
	}

	// 5. Validar que el pago se envió al repositorio de pagos
	if paymentsRepo.createdPayment == nil {
		t.Fatal("expected payment to be persisted via paymentsRepo")
	}
	if paymentsRepo.createdPayment.ID != result.ID {
		t.Errorf("expected persisted payment ID %s, got %s", result.ID, paymentsRepo.createdPayment.ID)
	}

	// 6. Validar que el evento Outbox se persistió con todos los campos requeridos
	if outboxRepo.createdEvent == nil {
		t.Fatal("expected outbox event to be persisted via outboxRepo")
	}

	event := outboxRepo.createdEvent
	if event.AggregateType != "PAYMENT" {
		t.Errorf("expected AggregateType 'PAYMENT', got %s", event.AggregateType)
	}
	if event.AggregateID != result.ID {
		t.Errorf("expected AggregateID %s, got %s", result.ID, event.AggregateID)
	}
	if event.EventType != "payment.created" {
		t.Errorf("expected EventType 'payment.created', got %s", event.EventType)
	}
	if event.Status != payment.OutboxStatusPending {
		t.Errorf("expected Status 'PENDING', got %s", event.Status)
	}

	// 7. Validar Payload JSON
	if len(event.Payload) == 0 {
		t.Error("expected non-empty outbox payload")
	}

	// 8. Validar TraceContext (propagación OTel W3C)
	if len(event.TraceContext) == 0 {
		t.Fatal("expected non-empty TraceContext in outbox event")
	}

	var traceMap map[string]string
	if err := json.Unmarshal(event.TraceContext, &traceMap); err != nil {
		t.Fatalf("failed to unmarshal TraceContext JSON: %v", err)
	}

	traceparent, ok := traceMap["traceparent"]
	if !ok || traceparent == "" {
		t.Fatalf("expected 'traceparent' in TraceContext, got map: %+v", traceMap)
	}

	// El formato W3C traceparent es: 00-{trace_id}-{span_id}-{flags}
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 || parts[0] != "00" {
		t.Errorf("invalid W3C traceparent format: %s", traceparent)
	}
}
