package domain

type AggregateType string
type OutboxStatus string
type PaymentStatus string
type EventType string

func (a AggregateType) ToString() string {
	return string(a)
}

func (e EventType) ToString() string {
	return string(e)
}

const (
	AggregateTypePayment AggregateType = "PAYMENT"
)

const (
	PaymentStatusCreated    PaymentStatus = "CREATED"
	PaymentStatusProcessing PaymentStatus = "PROCESSING"
	PaymentStatusSettled    PaymentStatus = "SETTLED"
	PaymentStatusFailed     PaymentStatus = "FAILED"
)

const (
	OutboxStatusPending    OutboxStatus = "PENDING"
	OutboxStatusProcessing OutboxStatus = "PROCESSING"
	OutboxStatusProcessed  OutboxStatus = "PROCESSED"
	OutboxStatusFailed     OutboxStatus = "FAILED"
)

const (
	EventTypePaymentCreated   EventType = "PAYMENT_CREATED"
	EventTypePaymentCompleted EventType = "PAYMENT_COMPLETED"
	EventTypePaymentFailed    EventType = "PAYMENT_FAILED"
)
