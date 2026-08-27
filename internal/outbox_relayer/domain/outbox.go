package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusProcessed OutboxStatus = "PROCESSED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

type OutboxEvent struct {
	ID            string         `gorm:"primaryKey;type:uuid"`
	AggregateType string         `gorm:"type:varchar(50);not null;index:idx_outbox_pending,priority:1"`
	AggregateID   string         `gorm:"type:varchar(64);not null"`
	Type          string         `gorm:"type:varchar(50);not null"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	Status        OutboxStatus   `gorm:"type:varchar(20);not null;default:'PENDING';index:idx_outbox_pending,priority:2"`
	CreatedAt     time.Time      `gorm:"not null;autoCreateTime;index:idx_outbox_pending,priority:3"`
	ProcessedAt   *time.Time     `gorm:"type:timestamptz"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}

// Factory para instanciar un nuevo evento antes de guardarlo en la DB
func NewOutboxEvent(aggregateType, aggregateID, eventType string, payloadData any) (*OutboxEvent, error) {
	payloadBytes, err := json.Marshal(payloadData)
	if err != nil {
		return nil, err
	}

	return &OutboxEvent{
		ID:            uuid.NewString(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Type:          eventType,
		Payload:       datatypes.JSON(payloadBytes),
		Status:        OutboxStatusPending,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
