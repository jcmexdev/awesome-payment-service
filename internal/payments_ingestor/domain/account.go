package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateAccountRequest struct {
	UserID   string `json:"user_id" binding:"required,uuid"`
	Currency string `json:"currency" binding:"required,len=3,alpha"`
	Balance  int64  `json:"balance" binding:"gte=0"`
}

func (r CreateAccountRequest) Validate() error {
	if r.UserID == "" {
		//return errors.NewAppError(errors.TypeValidationError)
	}
	panic("implement me")
}

func (r CreateAccountRequest) NewDomainAccount() *Account {
	return &Account{
		ID:        uuid.New().String(),
		UserID:    r.UserID,
		Currency:  r.Currency,
		Balance:   r.Balance,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

type Account struct {
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
	ID        string    `gorm:"primaryKey;type:varchar(255)"`
	UserID    string    `gorm:"index;type:varchar(255);not null"`
	Currency  string    `gorm:"type:varchar(10);not null"`
	Balance   int64     `gorm:"not null;default:0"`
	Version   int       `gorm:"not null;default:0"`
}
