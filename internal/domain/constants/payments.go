package constants

type PaymentStatus string

const (
	PaymentSuccess  PaymentStatus = "SUCCESS"
	PaymentFailed   PaymentStatus = "FAILED"
	PaymentCanceled PaymentStatus = "CANCELED"
	PaymentPending  PaymentStatus = "PENDING"
	PaymentCreated  PaymentStatus = "CREATED"
)
