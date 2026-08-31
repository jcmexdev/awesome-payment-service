package domain

import (
	"time"

	"gorm.io/datatypes"
)

type OutboxEvent struct {
	CreatedAt     time.Time      `gorm:"not null;autoCreateTime;index:idx_outbox_pending,priority:3"`
	LockedUntil   *time.Time     `gorm:"type:timestamptz"`
	ProcessedAt   *time.Time     `gorm:"type:timestamptz"`
	ID            string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AggregateType AggregateType  `gorm:"type:varchar(50);not null;index:idx_outbox_pending,priority:1"`
	AggregateID   string         `gorm:"type:varchar(64);not null"`
	EventType     EventType      `gorm:"type:varchar(50);not null"`
	Status        OutboxStatus   `gorm:"type:varchar(20);not null;default:'PENDING';index:idx_outbox_pending,priority:2"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	TraceContext  datatypes.JSON `gorm:"type:jsonb"`
	Attempts      int            `gorm:"type:int;not null;default:0"`
}

func NewOutboxEvent(
	ID string,
	aggregateType AggregateType,
	aggregateID string,
	eventType EventType,
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
