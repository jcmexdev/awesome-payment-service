package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/constants"
	"github.com/redis/go-redis/v9"
)

type IdempotencyCache struct {
	client *redis.Client
}

func NewIdempotencyCache(client *redis.Client) *IdempotencyCache {
	return &IdempotencyCache{client: client}
}

func (i IdempotencyCache) Lock(ctx context.Context, key string, ttl time.Duration) (bool, *domain.IdempotencyRecord, error) {
	reqID, _ := ctx.Value(constants.ContextKeyRequestID).(string)
	now := time.Now().UTC()
	lockRecord := domain.IdempotencyRecord{
		Key:       key,
		Status:    domain.IdempotencyStatusProcessing,
		RequestID: reqID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	lockVal, err := json.Marshal(lockRecord)
	if err != nil {
		return false, nil, fmt.Errorf("failed to marshal lock record: %w", err)
	}

	acquired, err := i.client.SetNX(ctx, key, lockVal, ttl).Result()

	if err != nil {
		return false, nil, fmt.Errorf("redis setnx execution failed: %w", err)
	}

	if acquired {
		return true, nil, nil
	}

	val, err := i.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("redis get execution failed: %w", err)
	}

	if val == domain.IdempotencyStatusProcessing {
		return false, &domain.IdempotencyRecord{
			Key:    key,
			Status: domain.IdempotencyStatusProcessing,
		}, nil
	}

	var record domain.IdempotencyRecord
	if err := json.Unmarshal([]byte(val), &record); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal record from redis: %w", err)
	}

	return false, &record, nil
}

func (i IdempotencyCache) Save(ctx context.Context, key string, statusCode int, body []byte, ttl time.Duration) error {
	now := time.Now().UTC()

	reqID, _ := ctx.Value(constants.ContextKeyRequestID).(string)

	record := domain.IdempotencyRecord{
		Key:          key,
		Status:       domain.IdempotencyStatusCompleted,
		ResponseCode: statusCode,
		ResponseBody: body,
		RequestID:    reqID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal idempotency record: %w", err)
	}

	if err := i.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error during save: %w", err)
	}

	return nil
}
