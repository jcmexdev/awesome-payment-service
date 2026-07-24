package gorm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LedgerRepository struct {
	db *gorm.DB
}

func NewLedgerRepository(db *gorm.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) CreateAccount(ctx context.Context, account *domain.Account) error {
	if err := account.Validate(); err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *LedgerRepository) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	var account domain.Account
	if err := r.db.WithContext(ctx).First(&account, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *LedgerRepository) ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
	if amountCents <= 0 {
		return errors.New("payment amount must be greater than zero")
	}
	if referenceID == "" {
		return errors.New("reference ID is required")
	}

	tr := otel.Tracer("payment-service")
	// Envolvemos la transacción ACID en un span explícito
	ctx, span := tr.Start(ctx, "payment.repository.execute_transaction")
	defer span.End()

	// GORM propaga automáticamente el contexto (ctx) a las operaciones internas del bloque
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account domain.Account

		// 1. Consulta de la cuenta usando Pessimistic Locking
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "id = ?", accountID).Error
		if err != nil {
			return fmt.Errorf("failed to locate account with pessimistic lock: %w", err)
		}

		// 2. Validación de saldo suficiente
		if account.CachedBalance < amountCents {
			return fmt.Errorf("insufficient balance: account balance is %d cents, requested payment is %d cents", account.CachedBalance, amountCents)
		}

		// 3. Registro inmutable del movimiento en LedgerEntry (Débito es negativo)
		entry := domain.LedgerEntry{
			ID:          uuid.New().String(),
			AccountID:   accountID,
			Amount:      -amountCents, // Negativo = Débito
			Type:        "PAYMENT",
			ReferenceID: referenceID,
			CreatedAt:   time.Now().UTC(),
		}
		if err := tx.Create(&entry).Error; err != nil {
			return fmt.Errorf("failed to register ledger entry (unique reference ID constraint violation): %w", err)
		}

		// 4. Actualización del balance e incremento de la versión
		account.CachedBalance -= amountCents
		account.Version += 1
		account.UpdatedAt = time.Now().UTC()

		if err := tx.Save(&account).Error; err != nil {
			return fmt.Errorf("failed to update account balance: %w", err)
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "transaction executed successfully")
	return nil
}
