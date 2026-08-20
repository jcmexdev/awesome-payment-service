package domain

import "time"

const (
	IdempotencyStatusProcessing = "PROCESSING"
	IdempotencyStatusCompleted  = "COMPLETED"
	IdempotencyStatusFailed     = "FAILED"
)

type IdempotencyRecord struct {
	Key          string    `gorm:"primaryKey;type:varchar(255)"`
	Status       string    `gorm:"type:varchar(50);not null"`
	ResponseCode int       `gorm:"not null"`
	ResponseBody []byte    `gorm:"type:bytea"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
	ExpiresAt    time.Time `gorm:"index;not null"`
}

func (r *IdempotencyRecord) IsProcessing() bool {
	return r.Status == IdempotencyStatusProcessing
}

func (r *IdempotencyRecord) IsCompleted() bool {
	return r.Status == IdempotencyStatusCompleted
}
