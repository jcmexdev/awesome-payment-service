package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jcmexdev/payment-service/internal/domain"
	_ "modernc.org/sqlite"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type IdempotencyCache struct {
	db *sql.DB
}

func NewConnection(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("path required")
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)
	db, err := otelsql.Open("sqlite", dsn,
		otelsql.WithAttributes(semconv.DBSystemSqlite),
		otelsql.WithDBName(path),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite ping failed: %w", err)
	}

	return db, nil
}

func NewIdempotencyCache(ctx context.Context, db *sql.DB) (*IdempotencyCache, error) {
	query := `
	CREATE TABLE IF NOT EXISTS idempotency_records (
       key TEXT PRIMARY KEY,
       status TEXT NOT NULL,
       response_code INTEGER,
       response_body BLOB,
       expires_at DATETIME NOT NULL,
       created_at DATETIME NOT NULL
    );
	CREATE INDEX IF NOT EXISTS idx_expires_at ON idempotency_records(expires_at);
	`

	execCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if _, err := db.ExecContext(execCtx, query); err != nil {
		return nil, fmt.Errorf("failed to create idempotency table in sqlite: %w", err)
	}

	return &IdempotencyCache{db: db}, nil
}

func (i IdempotencyCache) Lock(ctx context.Context, key string, ttl time.Duration) (bool, *domain.IdempotencyRecord, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	_, _ = i.db.ExecContext(ctx, "DELETE FROM idempotency_records WHERE expires_at < ?", now)

	insertQuery := `
		INSERT INTO idempotency_records (key, status, expires_at, created_at) 
		VALUES (?, ?, ?, ?);`

	_, err := i.db.ExecContext(ctx, insertQuery, key, domain.IdempotencyStatusProcessing, expiresAt, now)

	if err == nil {
		return true, nil, nil
	}

	selectQuery := `
		SELECT key, status, response_code, response_body, created_at 
		FROM idempotency_records 
		WHERE key = ?;`

	row := i.db.QueryRowContext(ctx, selectQuery, key)

	var rec domain.IdempotencyRecord
	var responseCode sql.NullInt64
	var responseBody []byte

	err = row.Scan(&rec.Key, &rec.Status, &responseCode, &responseBody, &rec.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("failed to query existing idempotency record in sqlite: %w", err)
	}

	if responseCode.Valid {
		rec.ResponseCode = int(responseCode.Int64)
	}
	rec.ResponseBody = responseBody

	return false, &rec, nil
}

func (i IdempotencyCache) Save(ctx context.Context, key string, statusCode int, body []byte, ttl time.Duration) error {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	query := `
		INSERT INTO idempotency_records (key, status, response_code, response_body, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			status = excluded.status,
			response_code = excluded.response_code,
			response_body = excluded.response_body,
			expires_at = excluded.expires_at;`

	_, err := i.db.ExecContext(ctx, query,
		key,
		domain.IdempotencyStatusCompleted,
		statusCode,
		body,
		expiresAt,
		now,
	)
	if err != nil {
		return fmt.Errorf("sqlite upsert failed: %w", err)
	}

	return nil
}
