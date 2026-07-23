package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIdempotencyCache_Save(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sqlite-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	ctx := context.Background()

	db, err := NewConnection(ctx, dbPath)
	if err != nil {
		t.Fatalf("failed to open connection: %v", err)
	}
	defer db.Close()

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
