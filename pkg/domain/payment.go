package domain

import (
	"errors"
	"fmt"
	"time"
)

type Payment struct {
	CreatedAt time.Time     `gorm:"not null"`
	UpdatedAt time.Time     `gorm:"not null"`
	ID        string        `gorm:"primaryKey;type:varchar(255)"`
	AccountID string        `gorm:"index;type:varchar(255);not null"`
	Currency  string        `gorm:"type:varchar(3);not null"`
	Status    PaymentStatus `gorm:"type:varchar(15);not null"`
	Amount    int64         `gorm:"not null;default:0"`
}

func NewPayment(id string, amount int64, currency, accountId string) (*Payment, error) {
	if id == "" || amount <= 0 {
		return nil, errors.New("datos de pago inválidos")
	}
	return &Payment{
		ID:        id,
		AccountID: accountId,
		Amount:    amount,
		Currency:  currency,
		Status:    PaymentStatusCreated,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (p *Payment) TransitionTo(event TransitionEvent) error {
	nextStatus, exists := allowedTransitions[p.Status][event]
	if !exists {
		return fmt.Errorf("%w: no se puede aplicar %s al estado %s", ErrInvalidTransitionEvent, event, p.Status)
	}

	p.Status = nextStatus
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (*Payment) TableName() string {
	return "payments"
}
