package domain

import (
	"errors"
	"time"
)

type Account struct {
	ID            string    `gorm:"primaryKey;type:varchar(255)"`
	UserID        string    `gorm:"index;type:varchar(255);not null"`
	Currency      string    `gorm:"type:varchar(10);not null"`
	CachedBalance int64     `gorm:"not null;default:0"`
	Version       int       `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

type LedgerEntry struct {
	ID        string    `gorm:"primaryKey;type:varchar(255)"`
	AccountID string    `gorm:"index;type:varchar(255);not null"`
	Amount    int64     `gorm:"not null"` // in cents: positive = credit, negative = debit
	Type      string    `gorm:"type:varchar(50);not null"`
	RequestId string    `gorm:"uniqueIndex;type:varchar(255);not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (a *Account) Validate() error {
	if a.ID == "" {
		return errors.New("account ID is required")
	}
	if a.UserID == "" {
		return errors.New("user ID is required")
	}
	if len(a.Currency) != 3 {
		return errors.New("currency must be a 3-character ISO-4217 code")
	}
	return nil
}

func (le *LedgerEntry) Validate() error {
	if le.ID == "" {
		return errors.New("ledger entry ID is required")
	}
	if le.AccountID == "" {
		return errors.New("ledger entry account ID is required")
	}
	if le.Amount == 0 {
		return errors.New("ledger entry amount cannot be zero")
	}
	if le.Type == "" {
		return errors.New("ledger entry type is required")
	}
	if le.RequestId == "" {
		return errors.New("ledger entry request ID is required")
	}
	return nil
}
