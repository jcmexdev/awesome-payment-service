package app

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/jcmexdev/payment-service/internal/infra/telemetry"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg               *config.Config
	Server            *http.Server
	shutdownTelemetry func(context.Context) error
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
		slog.Duration("IDEMPOTENCY_TTL", cfg.IdempotencyTTL),
	)

	// Initialize OpenTelemetry if configured
	var shutdownTelemetry func(context.Context) error
	if cfg.OtelCollectorAddr != "" {
		var err error
		shutdownTelemetry, err = telemetry.InitTracer(ctx, cfg.OtelServiceName, cfg.OtelCollectorAddr)
		if err != nil {
			slog.Error("Failed to initialize OpenTelemetry", "error", err)
		} else {
			slog.Info("OpenTelemetry initialized", "collector", cfg.OtelCollectorAddr, "service", cfg.OtelServiceName)
		}
	}

	redisClient, err := initRedis(ctx, cfg.RedisAddr)
	if err != nil {
		if shutdownTelemetry != nil {
			_ = shutdownTelemetry(ctx)
		}
		return nil, err
	}

	sqliteDB, err := initSqlite(ctx, cfg.IdempotencyFilePath)
	if err != nil {
		_ = redisClient.Close()
		if shutdownTelemetry != nil {
			_ = shutdownTelemetry(ctx)
		}
		return nil, err
	}

	redisCacheRepo := rediscache.NewIdempotencyCache(redisClient)

	sqliteCacheRepo, err := sqlite.NewIdempotencyCache(ctx, sqliteDB)
	if err != nil {
		_ = redisClient.Close()
		_ = sqliteDB.Close()
		if shutdownTelemetry != nil {
			_ = shutdownTelemetry(ctx)
		}
		return nil, err
	}
	idempotencyPersistence := cache.NewPersistenceCache(redisCacheRepo, sqliteCacheRepo, 50*time.Millisecond)
	idempMiddleware := middleware.NewIdempotencyMiddleware(idempotencyPersistence, cfg.IdempotencyTTL)

	r := router.NewRouter(
		router.WithHealthController(handler.NewHealthHandler()),
		router.WithPaymentsController(handler.NewPaymentsHandler()),
		router.WithIdempotencyMiddleware(idempMiddleware),
	)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	return &App{
		cfg:               cfg,
		Server:            srv,
		shutdownTelemetry: shutdownTelemetry,
	}, nil
}

func initSqlite(ctx context.Context, path string) (*sql.DB, error) {
	return sqlite.NewConnection(ctx, path)
}

func initRedis(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})

	client.AddHook(telemetry.NewOpenTelemetryRedisHook())

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

func (app *App) Shutdown(ctx context.Context) error {
	var errs []error

	if err := app.Server.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("server shutdown error: %w", err))
	}

	if app.shutdownTelemetry != nil {
		if err := app.shutdownTelemetry(ctx); err != nil {
			errs = append(errs, fmt.Errorf("telemetry shutdown error: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
