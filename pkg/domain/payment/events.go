package payment

import "time"

type PaymentStatus string

const (
	StatusCreated    PaymentStatus = "CREATED"
	StatusProcessing PaymentStatus = "PROCESSING"
	StatusSettled    PaymentStatus = "SETTLED"
	StatusFailed     PaymentStatus = "FAILED"
)

type TransitionEvent string

const (
	EventStartProcessing TransitionEvent = "START_PROCESSING"
	EventSettle          TransitionEvent = "SETTLED"
	EventFail            TransitionEvent = "FAILED"
)

// DTO del Evento para SQS / Stream
type CreatedEvent struct {
	PaymentID string    `json:"payment_id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}
