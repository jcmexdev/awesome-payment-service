package domain

import (
	"errors"
	"time"
)

type Payment struct {
	ID        string
	AccountID string
	Amount    int64
	Currency  string
	Status    string
	CreatedAt time.Time
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
