package infrastructure

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jcmexdev/payment-service/internal/outbox_relayer/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type OutboxRelayer struct {
	outboxRepo ports.OutboxRepository
	publisher  ports.MessagePublisher
	batchSize  int
	interval   time.Duration
}

func NewOutboxRelayer(
	outboxRepo ports.OutboxRepository,
	publisher ports.MessagePublisher,
	batchSize int,
	interval time.Duration,
) *OutboxRelayer {
	return &OutboxRelayer{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		batchSize:  batchSize,
		interval:   interval,
	}
}

func (r *OutboxRelayer) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	log.Println("[OutboxRelayer] Worker iniciado exitosamente...")

	for {
		select {
		case <-ctx.Done():
			log.Println("[OutboxRelayer] Apagando el worker de forma segura...")
			return
		case <-ticker.C:
			r.processPendingEvents(ctx)
		}
	}
}

func (r *OutboxRelayer) processPendingEvents(ctx context.Context) {
	maxAttempts := 3
	baseIntervalSeconds := 10
	events, err := r.outboxRepo.FetchAndLockPendingEvents(ctx, r.batchSize, baseIntervalSeconds, maxAttempts)
	if err != nil {
		log.Printf("[OutboxRelayer] Error al consultar eventos: %v\n", err)
		return
	}

	if len(events) == 0 {
		log.Printf("[OutboxRelayer] No eventos para procesar.\n")
		return
	}

	tr := otel.Tracer("outbox_relayer")

	for _, event := range events {
		// 1. Extraer y reconstruir el contexto distribuido de OTel desde TraceContext
		eventCtx := ctx
		carrier := propagation.MapCarrier{}

		if len(event.TraceContext) > 0 {
			if err := json.Unmarshal(event.TraceContext, &carrier); err == nil {
				eventCtx = otel.GetTextMapPropagator().Extract(ctx, carrier)
			}
		}

		// 2. Iniciar Child Span vinculado al TraceID original del Ingestor
		eventCtx, span := tr.Start(eventCtx, "outbox_relayer.publish_event",
			trace.WithSpanKind(trace.SpanKindProducer),
			trace.WithAttributes(
				attribute.String("messaging.system", "sqs"),
				attribute.String("messaging.destination", "payments-queue"),
				attribute.String("messaging.message_id", event.ID),
				attribute.String("aggregate.type", event.AggregateType.ToString()),
				attribute.String("aggregate.id", event.AggregateID),
				attribute.String("event.type", event.EventType.ToString()),
			),
		)

		// 3. Inyectar contexto activo en las cabeceras del mensaje hacia el Broker downstream
		outCarrier := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(eventCtx, outCarrier)

		outCarrier["event_id"] = event.ID
		outCarrier["event_type"] = event.EventType.ToString()

		msg := ports.Message{
			Destination: "payments-queue",
			Key:         event.AggregateID,
			Payload:     []byte(event.Payload),
			Headers:     outCarrier,
		}

		// 4. Publicar el mensaje en el Broker
		err := r.publisher.Publish(eventCtx, msg)
		if err != nil {
			log.Printf("[OutboxRelayer] Error enviando evento %s: %v\n", event.ID, err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			if event.Attempts == maxAttempts {
				_ = r.outboxRepo.MarkAsFailed(ctx, event.ID)
			}
			continue
		}

		span.SetStatus(codes.Ok, "event published successfully")
		span.End()

		if err := r.outboxRepo.MarkAsProcessed(ctx, event.ID); err != nil {
			log.Printf("[OutboxRelayer] Error actualizando estado %s: %v\n", event.ID, err)
		}
	}
}
