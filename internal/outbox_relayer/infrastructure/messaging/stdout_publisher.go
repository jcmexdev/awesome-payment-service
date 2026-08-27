package messaging

import (
	"context"
	"log"

	"github.com/jcmexdev/payment-service/internal/outbox_relayer/ports"
)

type StdoutPublisher struct{}

func NewStdoutPublisher() ports.MessagePublisher {
	return &StdoutPublisher{}
}

func (p *StdoutPublisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	log.Printf("[PUBLISH] Topic: %s | Key: %s | Payload: %s\n", topic, key, string(payload))
	return nil
}
