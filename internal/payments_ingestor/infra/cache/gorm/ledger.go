package gorm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domain2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	errors2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
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

func (r *LedgerRepository) CreateAccount(ctx context.Context, account *domain2.Account) error {
	if err := account.Validate(); err != nil {
		return errors2.NewAppError(constants.TypeValidationError, "INVALID_ACCOUNT", "invalid account validation failure", err).
			WithLogContext("user_id", account.UserID).
			WithLogContext("currency", account.Currency)
	}
	if err := r.db.WithContext(ctx).Create(account).Error; err != nil {
		return errors2.NewAppError(constants.TypeInternal, "DATABASE_ERROR", "failed to create account", err).
			WithLogContext("user_id", account.UserID).
			WithLogContext("currency", account.Currency)
	}
	return nil
}

func (r *LedgerRepository) GetAccount(ctx context.Context, id string) (*domain2.Account, error) {
	var account domain2.Account
	if err := r.db.WithContext(ctx).First(&account, "id = ?", id).Error; err != nil {
		if err == gormio.ErrRecordNotFound {
			return nil, errors2.NewAppError(constants.TypeNotFound, "ACCOUNT_NOT_FOUND", "account not found", err).
				WithLogContext("account_id", id)
		}
		return nil, errors2.NewAppError(constants.TypeInternal, "DATABASE_ERROR", "failed to retrieve account", err).
			WithLogContext("account_id", id)
	}
	return &account, nil
}

func (r *LedgerRepository) ProcessPayment(ctx context.Context, accountID string, amountCents int64, referenceID string) error {
	if amountCents <= 0 {
		return errors2.NewAppError(constants.TypeValidationError, "INVALID_PAYMENT", "payment amount must be greater than zero", nil).
			WithLogContext("account_id", accountID).
			WithLogContext("amount_cents", amountCents)
	}
	if referenceID == "" {
		return errors2.NewAppError(constants.TypeValidationError, "INVALID_PAYMENT", "reference ID is required", nil).
			WithLogContext("account_id", accountID)
	}

	tr := otel.Tracer("payments_ingestor")
	ctx, span := tr.Start(ctx, "payment.repository.execute_transaction")
	defer span.End()

	err := r.db.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		var account domain2.Account

		// 1. Consulta de la cuenta usando Pessimistic Locking
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "id = ?", accountID).Error
		if err != nil {
			if err == gormio.ErrRecordNotFound {
				return errors2.NewAppError(constants.TypeNotFound, "ACCOUNT_NOT_FOUND", "failed to locate account with pessimistic lock: record not found", err).
					WithLogContext("account_id", accountID)
			}
			return errors2.NewAppError(constants.TypeInternal, "DATABASE_ERROR", "failed to locate account with pessimistic lock", err).
				WithLogContext("account_id", accountID)
		}

		// 2. Validación de saldo suficiente
		if account.Balance < amountCents {
			return errors2.NewAppError(constants.TypeValidationError, "INSUFFICIENT_BALANCE", fmt.Sprintf("insufficient balance: account balance is %d cents, requested payment is %d cents", account.Balance, amountCents), nil).
				WithLogContext("account_id", accountID).
				WithLogContext("cached_balance", account.Balance).
				WithLogContext("requested_amount", amountCents)
		}

		// 3. Registro inmutable del movimiento en LedgerEntry (Débito es negativo)
		entry := domain2.LedgerEntry{
			ID:        uuid.New().String(),
			AccountID: accountID,
			Amount:    -amountCents,
			Type:      "PAYMENT",
			RequestId: referenceID,
			CreatedAt: time.Now().UTC(),
		}
		if err := tx.Create(&entry).Error; err != nil {
			return errors2.NewAppError(constants.TypeValidationError, "PAYMENT_DUPLICATED", "failed to register ledger entry (unique reference ID constraint violation)", err).
				WithLogContext("account_id", accountID).
				WithLogContext("reference_id", referenceID)
		}

		// 4. Actualización del balance e incremento de la versión
		account.Balance -= amountCents
		account.Version += 1
		account.UpdatedAt = time.Now().UTC()

		if err := tx.Save(&account).Error; err != nil {
			return errors2.NewAppError(constants.TypeInternal, "DATABASE_ERROR", "failed to update account balance", err).
				WithLogContext("account_id", accountID).
				WithLogContext("cached_balance", account.Balance)
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
