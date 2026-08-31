package ports

import "context"

type MessageConsumer interface {
	Start(ctx context.Context, workers int) error
}
