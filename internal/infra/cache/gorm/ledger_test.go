package gorm

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	// Set cache=shared and limit open connections to 1 to serialize database access
	// and prevent SQLite "database table is locked" errors during concurrent tests.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := db.AutoMigrate(&domain.Account{}, &domain.LedgerEntry{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	cleanup := func() {
		_ = sqlDB.Close()
	}

	return db, cleanup
}

func TestLedgerRepository_ProcessPayment_Success(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewLedgerRepository(db)
	ctx := context.Background()

	account := domain.Account{
		ID:            uuid.New().String(),
		UserID:        "user-123",
		Currency:      "USD",
		CachedBalance: 10000, // 100.00 USD
		Version:       0,
	}

	if err := repo.CreateAccount(ctx, &account); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	refID := "ref-payment-1"
	err := repo.ProcessPayment(ctx, account.ID, 3000, refID) // 30.00 USD payment
	if err != nil {
		t.Fatalf("ProcessPayment failed: %v", err)
	}

	// Verify account balance and version
	updatedAccount, err := repo.GetAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if updatedAccount.CachedBalance != 7000 {
		t.Errorf("expected balance 7000, got %d", updatedAccount.CachedBalance)
	}

	if updatedAccount.Version != 1 {
		t.Errorf("expected version 1, got %d", updatedAccount.Version)
	}

	// Verify ledger entry
	var entry domain.LedgerEntry
	if err := db.First(&entry, "reference_id = ?", refID).Error; err != nil {
		t.Fatalf("failed to find ledger entry: %v", err)
	}

	if entry.Amount != -3000 {
		t.Errorf("expected amount -3000, got %d", entry.Amount)
	}

	if entry.AccountID != account.ID {
		t.Errorf("expected account ID %s, got %s", account.ID, entry.AccountID)
	}
}

func TestLedgerRepository_ProcessPayment_InsufficientBalance(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewLedgerRepository(db)
	ctx := context.Background()

	account := domain.Account{
		ID:            uuid.New().String(),
		UserID:        "user-123",
		Currency:      "USD",
		CachedBalance: 1000, // 10.00 USD
		Version:       0,
	}

	if err := repo.CreateAccount(ctx, &account); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	refID := "ref-payment-insufficient"
	err := repo.ProcessPayment(ctx, account.ID, 3000, refID) // 30.00 USD payment
	if err == nil {
		t.Fatal("expected error due to insufficient balance, got nil")
	}

	// Verify balance is unchanged and no ledger entry was created
	updatedAccount, err := repo.GetAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if updatedAccount.CachedBalance != 1000 {
		t.Errorf("expected balance to remain 1000, got %d", updatedAccount.CachedBalance)
	}

	var count int64
	db.Model(&domain.LedgerEntry{}).Where("reference_id = ?", refID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 ledger entries, got %d", count)
	}
}

func TestLedgerRepository_ProcessPayment_DbIdempotency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewLedgerRepository(db)
	ctx := context.Background()

	account := domain.Account{
		ID:            uuid.New().String(),
		UserID:        "user-123",
		Currency:      "USD",
		CachedBalance: 10000, // 100.00 USD
		Version:       0,
	}

	if err := repo.CreateAccount(ctx, &account); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	refID := "ref-payment-duplicate"
	err := repo.ProcessPayment(ctx, account.ID, 2000, refID)
	if err != nil {
		t.Fatalf("first payment failed: %v", err)
	}

	// Attempt second payment with duplicate ref ID
	err = repo.ProcessPayment(ctx, account.ID, 3000, refID)
	if err == nil {
		t.Fatal("expected error due to duplicate reference ID, got nil")
	}

	// Verify balance only decreased once
	updatedAccount, err := repo.GetAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if updatedAccount.CachedBalance != 8000 {
		t.Errorf("expected balance to be 8000, got %d", updatedAccount.CachedBalance)
	}
}

func TestLedgerRepository_ProcessPayment_Concurrency(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewLedgerRepository(db)
	ctx := context.Background()

	account := domain.Account{
		ID:            uuid.New().String(),
		UserID:        "user-123",
		Currency:      "USD",
		CachedBalance: 10000, // 100.00 USD
		Version:       0,
	}

	if err := repo.CreateAccount(ctx, &account); err != nil {
		t.Fatalf("failed to create account: %v", err)
	}

	var wg sync.WaitGroup
	numRequests := 5
	errs := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			refID := uuid.New().String()
			// Each request deducts 1000 cents (10.00 USD)
			err := repo.ProcessPayment(ctx, account.ID, 1000, refID)
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent payment failed: %v", err)
	}

	// Verify balance is exactly 5000 (10000 - 5 * 1000)
	updatedAccount, err := repo.GetAccount(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if updatedAccount.CachedBalance != 5000 {
		t.Errorf("expected balance 5000, got %d", updatedAccount.CachedBalance)
	}

	if updatedAccount.Version != 5 {
		t.Errorf("expected version 5, got %d", updatedAccount.Version)
	}
}
