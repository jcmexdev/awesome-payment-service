package domain

import "time"

type TransitionEvent string

const (
	EventStartProcessing TransitionEvent = "START_PROCESSING"
	EventSettle          TransitionEvent = "SETTLED"
	EventFail            TransitionEvent = "FAILED"
)

type CreatedEvent struct {
	CreatedAt time.Time `json:"created_at"`
	PaymentID string    `json:"payment_id"`
	Currency  string    `json:"currency"`
	Amount    int64     `json:"amount"`
}
