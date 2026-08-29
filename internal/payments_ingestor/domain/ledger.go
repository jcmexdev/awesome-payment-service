package domain

import (
	"errors"
	"time"
)

type LedgerEntry struct {
	CreatedAt time.Time `gorm:"not null"`
	ID        string    `gorm:"primaryKey;type:varchar(255)"`
	AccountID string    `gorm:"index;type:varchar(255);not null"`
	Type      string    `gorm:"type:varchar(50);not null"`
	RequestId string    `gorm:"uniqueIndex;type:varchar(255);not null"`
	Amount    int64     `gorm:"not null"`
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
