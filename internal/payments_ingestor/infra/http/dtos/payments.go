package dtos

import (
	"time"

	"github.com/jcmexdev/payment-service/pkg/domain"
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

func PaymentFromDomain(p *domain.Payment) PaymentResponseDTO {
	return PaymentResponseDTO{
		PaymentID: p.ID,
		Amount:    p.Amount,
		Currency:  p.Currency,
		Status:    string(domain.PaymentStatusCreated),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
	}
}
