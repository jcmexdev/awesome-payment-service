package payment

import (
	"time"

	"gorm.io/datatypes"
)

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "PENDING"
	OutboxStatusProcessed OutboxStatus = "PROCESSED"
	OutboxStatusFailed    OutboxStatus = "FAILED"
)

type OutboxEvent struct {
	ID            string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AggregateType string         `gorm:"type:varchar(50);not null;index:idx_outbox_pending,priority:1"`
	AggregateID   string         `gorm:"type:varchar(64);not null"`
	EventType     string         `gorm:"type:varchar(50);not null"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	TraceContext  datatypes.JSON `gorm:"type:jsonb"`
	Status        OutboxStatus   `gorm:"type:varchar(20);not null;default:'PENDING';index:idx_outbox_pending,priority:2"`
	CreatedAt     time.Time      `gorm:"not null;autoCreateTime;index:idx_outbox_pending,priority:3"`
	ProcessedAt   *time.Time     `gorm:"type:timestamptz"`
}

func NewOutboxEvent(
	ID string,
	aggregateType string,
	aggregateID string,
	eventType string,
	payload datatypes.JSON,
	traceContext datatypes.JSON,
	status OutboxStatus,
) *OutboxEvent {
	return &OutboxEvent{
		ID:            ID,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		TraceContext:  traceContext,
		Status:        status,
		CreatedAt:     time.Now().UTC(),
	}
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}
