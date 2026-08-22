package dtos

import (
	"time"

	"github.com/jcmexdev/payment-service/internal/domain"
)

type CreateAccountResponse struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	Currency      string    `json:"currency"`
	CachedBalance int64     `json:"balance"`
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func NewCreateAccountResponse(account *domain.Account) *CreateAccountResponse {
	return &CreateAccountResponse{
		ID:            account.ID,
		UserID:        account.UserID,
		Currency:      account.Currency,
		CachedBalance: account.CachedBalance,
		Version:       account.Version,
		CreatedAt:     account.CreatedAt,
		UpdatedAt:     account.UpdatedAt,
	}
}
