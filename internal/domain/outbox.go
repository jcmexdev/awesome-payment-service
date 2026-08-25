package domain

import (
	"time"

	"gorm.io/datatypes"
)

type OutboxEvent struct {
	ID            string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AggregateType string         `gorm:"type:varchar(50);not null;index:idx_outbox_pending,priority:1"`
	AggregateID   string         `gorm:"type:varchar(64);not null"`
	Type          string         `gorm:"type:varchar(50);not null"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	Status        string         `gorm:"type:varchar(20);not null;default:'PENDING';index:idx_outbox_pending,priority:2"`
	CreatedAt     time.Time      `gorm:"not null;autoCreateTime;index:idx_outbox_pending,priority:3"`
	ProcessedAt   *time.Time     `gorm:"type:timestamptz"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}
