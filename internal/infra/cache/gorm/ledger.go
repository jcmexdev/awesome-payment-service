package gorm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	gormio "gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LedgerRepository struct {
	db *gormio.DB
}

func NewLedgerRepository(db *gormio.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) CreateAccount(ctx context.Context, account *domain.Account) error {
	if err := account.Validate(); err != nil {
		return errors.NewAppError(errors.TypeValidationError, "INVALID_ACCOUNT", "invalid account validation failure", err).
			WithContext("user_id", account.UserID).
			WithContext("currency", account.Currency)
	}
	if err := r.db.WithContext(ctx).Create(account).Error; err != nil {
		return errors.NewAppError(errors.TypeInternal, "DATABASE_ERROR", "failed to create account", err).
			WithContext("user_id", account.UserID).
			WithContext("currency", account.Currency)
	}
	return nil
}

func (r *LedgerRepository) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	var account domain.Account
	if err := r.db.WithContext(ctx).First(&account, "id = ?", id).Error; err != nil {
		if err == gormio.ErrRecordNotFound {
			return nil, errors.NewAppError(errors.TypeNotFound, "ACCOUNT_NOT_FOUND", "account not found", err).
				WithContext("account_id", id)
		}
		return nil, errors.NewAppError(errors.TypeInternal, "DATABASE_ERROR", "failed to retrieve account", err).
			WithContext("account_id", id)
	}
	return &account, nil
}

func (r *LedgerRepository) ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
	if amountCents <= 0 {
		return errors.NewAppError(errors.TypeValidationError, "INVALID_PAYMENT", "payment amount must be greater than zero", nil).
			WithContext("account_id", accountID).
			WithContext("amount_cents", amountCents)
	}
	if referenceID == "" {
		return errors.NewAppError(errors.TypeValidationError, "INVALID_PAYMENT", "reference ID is required", nil).
			WithContext("account_id", accountID)
	}

	tr := otel.Tracer("payment-service")
	ctx, span := tr.Start(ctx, "payment.repository.execute_transaction")
	defer span.End()

	err := r.db.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		var account domain.Account

		// 1. Consulta de la cuenta usando Pessimistic Locking
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "id = ?", accountID).Error
		if err != nil {
			if err == gormio.ErrRecordNotFound {
				return errors.NewAppError(errors.TypeNotFound, "ACCOUNT_NOT_FOUND", "failed to locate account with pessimistic lock: record not found", err).
					WithContext("account_id", accountID)
			}
			return errors.NewAppError(errors.TypeInternal, "DATABASE_ERROR", "failed to locate account with pessimistic lock", err).
				WithContext("account_id", accountID)
		}

		// 2. Validación de saldo suficiente
		if account.CachedBalance < amountCents {
			return errors.NewAppError(errors.TypeValidationError, "INSUFFICIENT_BALANCE", fmt.Sprintf("insufficient balance: account balance is %d cents, requested payment is %d cents", account.CachedBalance, amountCents), nil).
				WithContext("account_id", accountID).
				WithContext("cached_balance", account.CachedBalance).
				WithContext("requested_amount", amountCents)
		}

		// 3. Registro inmutable del movimiento en LedgerEntry (Débito es negativo)
		entry := domain.LedgerEntry{
			ID:          uuid.New().String(),
			AccountID:   accountID,
			Amount:      -amountCents,
			Type:        "PAYMENT",
			ReferenceID: referenceID,
			CreatedAt:   time.Now().UTC(),
		}
		if err := tx.Create(&entry).Error; err != nil {
			return errors.NewAppError(errors.TypeValidationError, "PAYMENT_DUPLICATED", "failed to register ledger entry (unique reference ID constraint violation)", err).
				WithContext("account_id", accountID).
				WithContext("reference_id", referenceID)
		}

		// 4. Actualización del balance e incremento de la versión
		account.CachedBalance -= amountCents
		account.Version += 1
		account.UpdatedAt = time.Now().UTC()

		if err := tx.Save(&account).Error; err != nil {
			return errors.NewAppError(errors.TypeInternal, "DATABASE_ERROR", "failed to update account balance", err).
				WithContext("account_id", accountID).
				WithContext("cached_balance", account.CachedBalance)
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
