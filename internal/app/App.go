package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jcmexdev/payment-service/internal/config"
	"github.com/jcmexdev/payment-service/internal/infra/cache"
	rediscache "github.com/jcmexdev/payment-service/internal/infra/cache/redis"
	"github.com/jcmexdev/payment-service/internal/infra/cache/sqlite"
	router "github.com/jcmexdev/payment-service/internal/infra/http"
	"github.com/jcmexdev/payment-service/internal/infra/http/handler"
	"github.com/jcmexdev/payment-service/internal/infra/http/middleware"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg    *config.Config
	Server *http.Server
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Configuration loaded successfully",
		slog.String("PORT", cfg.Port),
		slog.String("REDIS_ADDR", cfg.RedisAddr),
		slog.Duration("REDIS_TIMEOUT", cfg.RedisTimeout),
		slog.String("LOG_LEVEL", cfg.LogLevel),
		slog.String("IDEMPOTENCY_FILE_PATH", cfg.IdempotencyFilePath),
	)

	redisClient, err := initRedis(ctx, cfg.RedisAddr)
	if err != nil {
		return nil, err
	}

	sqliteDB, err := initSqlite(ctx, cfg.IdempotencyFilePath)
	if err != nil {
		return nil, err
	}

	redisCacheRepo := rediscache.NewIdempotencyCache(redisClient)

	sqliteCacheRepo, err := sqlite.NewIdempotencyCache(ctx, sqliteDB)
	if err != nil {
		return nil, err
	}
	idempotencyPersistence := cache.NewPersistenceCache(redisCacheRepo, sqliteCacheRepo, 50*time.Millisecond)
	idempMiddleware := middleware.NewIdempotencyMiddleware(idempotencyPersistence, 50*time.Millisecond)

	r := router.NewRouter(
		router.WithHealthController(handler.NewHealthHandler()),
		router.WithPaymentsController(handler.NewPaymentsHandler()),
		router.WithIdempotencyMiddleware(idempMiddleware),
	)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	return &App{cfg: cfg, Server: srv}, nil
}

func initSqlite(ctx context.Context, path string) (*sql.DB, error) {
	return sqlite.NewConnection(ctx, path)
}

func initRedis(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed at %s: %w", addr, err)
	}

	return client, nil
}

func (app *App) Run() error {
	return app.Server.ListenAndServe()
}
