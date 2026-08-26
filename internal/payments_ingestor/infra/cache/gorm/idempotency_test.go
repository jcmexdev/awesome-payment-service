package gorm

import (
	"context"
	"testing"
	"time"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIdempotencyCache_Save(t *testing.T) {
	// Use in-memory SQLite for testing to avoid database server dependencies
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(&domain.IdempotencyRecord{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	ctx := context.Background()
	cache, err := NewIdempotencyCache(ctx, db)
	if err != nil {
		t.Fatalf("failed to create idempotency cache: %v", err)
	}

	key := "test-key-abc"
	statusCode := 201
	body := []byte(`{"status":"created"}`)
	ttl := time.Hour

	// Test Save
	err = cache.Save(ctx, key, statusCode, body, ttl)
	if err != nil {
		t.Fatalf("failed to save record: %v", err)
	}

	// Test Lock (which also queries existing record)
	acquired, record, err := cache.Lock(ctx, key, ttl)
	if err != nil {
		t.Fatalf("failed to lock: %v", err)
	}

	if acquired {
		t.Error("expected acquired to be false, got true")
	}

	if record == nil {
		t.Fatal("expected record to be found, got nil")
	}

	if record.Key != key {
		t.Errorf("expected key %s, got %s", key, record.Key)
	}

	if record.ResponseCode != statusCode {
		t.Errorf("expected response code %d, got %d", statusCode, record.ResponseCode)
	}

	if string(record.ResponseBody) != string(body) {
		t.Errorf("expected response body %s, got %s", string(body), string(record.ResponseBody))
	}
}
