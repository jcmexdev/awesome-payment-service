package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jcmexdev/payment-service/internal/domain"
)

type mockLedgerRepo struct {
	processPaymentFunc func(ctx context.Context, accountID string, amountCents int64, referenceID string) error
}

func (m *mockLedgerRepo) CreateAccount(ctx context.Context, account *domain.Account) error {
	return nil
}

func (m *mockLedgerRepo) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	return nil, nil
}

func (m *mockLedgerRepo) ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
	return m.processPaymentFunc(ctx, accountID, amountCents, referenceID)
}

func TestPaymentUseCase_ProcessPayment_Success(t *testing.T) {
	repo := &mockLedgerRepo{
		processPaymentFunc: func(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
			return nil
		},
	}

	uc := NewPaymentUseCase(repo)
	err := uc.ProcessPayment(context.Background(), "acc-123", 1000, "ref-123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestPaymentUseCase_ProcessPayment_Error(t *testing.T) {
	repo := &mockLedgerRepo{
		processPaymentFunc: func(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
			return errors.New("insufficient balance")
		},
	}

	uc := NewPaymentUseCase(repo)
	err := uc.ProcessPayment(context.Background(), "acc-123", 1000, "ref-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "insufficient balance" {
		t.Errorf("expected 'insufficient balance', got '%v'", err.Error())
	}
}
