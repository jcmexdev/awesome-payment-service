package dtos

import (
	"time"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
)

type PaymentRequestDTO struct {
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
}

type PaymentResponseDTO struct {
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func PaymentFromDomain(payment *domain.Payment) PaymentResponseDTO {
	return PaymentResponseDTO{
		PaymentID: payment.ID,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		Status:    payment.Status,
		CreatedAt: payment.CreatedAt.UTC().Format(time.RFC3339),
	}
}
