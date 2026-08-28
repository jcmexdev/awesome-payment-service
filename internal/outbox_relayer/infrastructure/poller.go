package infrastructure

import (
	"context"
	"log"
	"time"

	"github.com/jcmexdev/payment-service/internal/outbox_relayer/ports"
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
	events, err := r.outboxRepo.FetchPendingEvents(ctx, r.batchSize)
	if err != nil {
		log.Printf("[OutboxRelayer] Error al consultar eventos: %v\n", err)
		return
	}

	if len(events) == 0 {
		log.Printf("[OutboxRelayer] No eventos para procesar.\n")
		return
	}

	for _, event := range events {
		msg := ports.Message{
			Payload: []byte(event.Payload),
		}
		err := r.publisher.Publish(ctx, msg)

		if err != nil {
			log.Printf("[OutboxRelayer] Error enviando evento %s: %v\n", event.ID, err)
			_ = r.outboxRepo.MarkAsFailed(ctx, event.ID)
			continue
		}

		if err := r.outboxRepo.MarkAsProcessed(ctx, event.ID); err != nil {
			log.Printf("[OutboxRelayer] Error actualizando estado %s: %v\n", event.ID, err)
		}
	}
}
