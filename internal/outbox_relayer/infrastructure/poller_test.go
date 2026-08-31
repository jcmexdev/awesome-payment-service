package infrastructure_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jcmexdev/payment-service/internal/outbox_relayer/infrastructure"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/ports"
	"github.com/jcmexdev/payment-service/pkg/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type mockOutboxRelayRepo struct {
	events       []domain.OutboxEvent
	processedIDs []string
	failedIDs    []string
}

func (m *mockOutboxRelayRepo) FetchAndLockPendingEvents(ctx context.Context, batchSize int, baseIntervalSeconds int, maxAttempts int) ([]domain.OutboxEvent, error) {
	return m.events, nil
}

func (m *mockOutboxRelayRepo) FetchPendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	return m.events, nil
}

func (m *mockOutboxRelayRepo) MarkAsProcessed(ctx context.Context, eventID string) error {
	m.processedIDs = append(m.processedIDs, eventID)
	return nil
}

func (m *mockOutboxRelayRepo) MarkAsFailed(ctx context.Context, eventID string) error {
	m.failedIDs = append(m.failedIDs, eventID)
	return nil
}

type mockPublisher struct {
	publishedMsgs []ports.Message
}

func (m *mockPublisher) Publish(ctx context.Context, msg ports.Message) error {
	m.publishedMsgs = append(m.publishedMsgs, msg)
	return nil
}

func TestOutboxRelayer_ExtractsTraceContextAndPropagates(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Contexto simulado del Ingestor
	originalTraceParent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	traceMap := map[string]string{
		"traceparent": originalTraceParent,
	}
	traceBytes, _ := json.Marshal(traceMap)

	event := domain.OutboxEvent{
		ID:            "event-123",
		AggregateType: "PAYMENT",
		AggregateID:   "pay-abc",
		EventType:     "payment.created",
		Payload:       []byte(`{"id":"pay-abc","amount":1000}`),
		TraceContext:  traceBytes,
		Status:        domain.OutboxStatusPending,
		CreatedAt:     time.Now().UTC(),
	}

	repo := &mockOutboxRelayRepo{
		events: []domain.OutboxEvent{event},
	}
	publisher := &mockPublisher{}

	relayer := infrastructure.NewOutboxRelayer(repo, publisher, 10, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go relayer.Start(ctx)

	// Esperar un ciclo del poller
	time.Sleep(150 * time.Millisecond)
	cancel()

	// 1. Validar que el mensaje fue publicado
	if len(publisher.publishedMsgs) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(publisher.publishedMsgs))
	}

	msg := publisher.publishedMsgs[0]
	if msg.Key != "pay-abc" {
		t.Errorf("expected Key 'pay-abc', got %s", msg.Key)
	}

	// 2. Validar que las cabeceras contienen el traceparent con el mismo TraceID original
	if msg.Headers == nil {
		t.Fatal("expected non-nil headers in published message")
	}

	publishedTraceparent, ok := msg.Headers["traceparent"]
	if !ok || publishedTraceparent == "" {
		t.Fatalf("expected 'traceparent' in published headers, got: %+v", msg.Headers)
	}

	// El TraceID debe ser exactamente el mismo que el original (4bf92f3577b34da6a3ce929d0e0e4736)
	originalTraceID := strings.Split(originalTraceParent, "-")[1]
	publishedTraceID := strings.Split(publishedTraceparent, "-")[1]

	if publishedTraceID != originalTraceID {
		t.Errorf("expected TraceID %s to be preserved across distributed boundary, got %s", originalTraceID, publishedTraceID)
	}

	// 3. Validar que fue marcado como PROCESSED
	if len(repo.processedIDs) != 1 || repo.processedIDs[0] != "event-123" {
		t.Errorf("expected event 'event-123' to be marked as processed, got %v", repo.processedIDs)
	}
}
