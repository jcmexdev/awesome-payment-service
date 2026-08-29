package domain

import "time"

const (
	IdempotencyStatusProcessing = "PROCESSING"
	IdempotencyStatusCompleted  = "COMPLETED"
	IdempotencyStatusFailed     = "FAILED"
)

type IdempotencyRecord struct {
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
	ExpiresAt    time.Time `gorm:"index;not null"`
	Key          string    `gorm:"primaryKey;type:varchar(255)"`
	Status       string    `gorm:"type:varchar(50);not null"`
	RequestID    string    `gorm:"type:varchar(255)"`
	ResponseBody []byte    `gorm:"type:bytea"`
	ResponseCode int       `gorm:"not null"`
}

func (r *IdempotencyRecord) IsProcessing() bool {
	return r.Status == IdempotencyStatusProcessing
}

func (r *IdempotencyRecord) IsCompleted() bool {
	return r.Status == IdempotencyStatusCompleted
}
