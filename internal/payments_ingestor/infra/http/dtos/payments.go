package dtos

import (
	"time"

	"github.com/jcmexdev/payment-service/pkg/domain/payment"
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

func PaymentFromDomain(p *payment.Payment) PaymentResponseDTO {
	return PaymentResponseDTO{
		PaymentID: p.ID,
		Amount:    p.Amount,
		Currency:  p.Currency,
		Status:    string(payment.StatusCreated),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
	}
}
