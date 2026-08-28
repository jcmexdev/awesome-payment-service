// internal/outbox_relayer/ports/message_publisher.go
package ports

import "context"

type Message struct {
	Destination string
	Key         string
	Payload     []byte
	Headers     map[string]string
}

type MessagePublisher interface {
	Publish(ctx context.Context, msg Message) error
}
