// internal/outbox_relayer/ports/message_publisher.go
package ports

import "context"

type MessagePublisher interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
}
