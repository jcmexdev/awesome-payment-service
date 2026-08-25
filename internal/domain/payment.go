package domain

import (
	"errors"
	"time"
)

type AuthorizePaymentRequest struct {
	AccountID string `json:"account_id" binding:"required,uuid"`
	Amount    int64  `json:"amount" binding:"required,gt=0"`
	Currency  string `json:"currency" binding:"required,len=3,alpha"`
}

type Payment struct {
	ID        string    `gorm:"primaryKey;type:varchar(255)"`
	AccountID string    `gorm:"index;type:varchar(255);not null"`
	Amount    int64     `gorm:"not null;default:0"`
	Currency  string    `gorm:"type:varchar(3);not null"`
	Status    string    `gorm:"type:varchar(15);not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (p *Payment) Validate() error {
	if p.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if p.Currency == "" {
		return errors.New("currency is required")
	}
	if p.AccountID == "" {
		return errors.New("account_id is required")
	}
	return nil
}
