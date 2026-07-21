package domain

import "time"

const (
	IdempotencyStatusProcessing = "PROCESSING"
	IdempotencyStatusCompleted  = "COMPLETED"
	IdempotencyStatusFailed     = "FAILED"
)

type IdempotencyRecord struct {
	Key          string
	Status       string
	ResponseCode int
	ResponseBody []byte
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (r *IdempotencyRecord) IsProcessing() bool {
	return r.Status == IdempotencyStatusProcessing
}

func (r *IdempotencyRecord) IsCompleted() bool {
	return r.Status == IdempotencyStatusCompleted
}
