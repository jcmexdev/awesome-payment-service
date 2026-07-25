package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jcmexdev/payment-service/internal/app/usecase"
	"github.com/jcmexdev/payment-service/internal/config"
	"github.com/jcmexdev/payment-service/internal/infra/cache"
	"github.com/jcmexdev/payment-service/internal/infra/cache/gorm"
	rediscache "github.com/jcmexdev/payment-service/internal/infra/cache/redis"
	router "github.com/jcmexdev/payment-service/internal/infra/http"
	"github.com/jcmexdev/payment-service/internal/infra/http/handler"
	"github.com/jcmexdev/payment-service/internal/infra/http/middleware"
	"github.com/jcmexdev/payment-service/internal/infra/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/redis/go-redis/v9"
	gormio "gorm.io/gorm"
)

type App struct {
	cfg               *config.Config
	Server            *http.Server
	db                *gormio.DB
	redisClient       *redis.Client
	shutdownTelemetry func(context.Context) error
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(telemetry.NewContextHandler(jsonHandler))
	slog.SetDefault(logger)

	slog.Info("Configuration loaded successfully",
		slog.String("PORT", cfg.Port),
		slog.String("REDIS_ADDR", cfg.RedisAddr),
		slog.Duration("REDIS_TIMEOUT", cfg.RedisTimeout),
		slog.String("LOG_LEVEL", cfg.LogLevel),
		slog.String("DATABASE_URL", cfg.DatabaseURL),
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

	gormDB, err := initGorm(ctx, cfg.DatabaseURL)
	if err != nil {
		_ = redisClient.Close()
		if shutdownTelemetry != nil {
			_ = shutdownTelemetry(ctx)
		}
		return nil, err
	}

	// Register DB stats collector for Prometheus
	if sqlDB, err := gormDB.DB(); err == nil {
		prometheus.MustRegister(collectors.NewDBStatsCollector(sqlDB, "payment_db"))
		slog.Info("Prometheus DBStatsCollector registered successfully")
	}

	redisCacheRepo := rediscache.NewIdempotencyCache(redisClient)

	gormCacheRepo, err := gorm.NewIdempotencyCache(ctx, gormDB)
	if err != nil {
		_ = redisClient.Close()
		if sqlDB, err := gormDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		if shutdownTelemetry != nil {
			_ = shutdownTelemetry(ctx)
		}
		return nil, err
	}
	idempotencyPersistence := cache.NewPersistenceCache(redisCacheRepo, gormCacheRepo, 50*time.Millisecond)
	idempMiddleware := middleware.NewIdempotencyMiddleware(idempotencyPersistence, cfg.IdempotencyTTL)

	ledgerRepo := gorm.NewLedgerRepository(gormDB)
	paymentUseCase := usecase.NewPaymentUseCase(ledgerRepo)

	r := router.NewRouter(
		router.WithHealthController(handler.NewHealthHandler()),
		router.WithPaymentsController(handler.NewPaymentsHandler(paymentUseCase)),
		router.WithIdempotencyMiddleware(idempMiddleware),
	)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	return &App{
		cfg:               cfg,
		Server:            srv,
		db:                gormDB,
		redisClient:       redisClient,
		shutdownTelemetry: shutdownTelemetry,
	}, nil
}

func initGorm(ctx context.Context, dsn string) (*gormio.DB, error) {
	return gorm.NewConnection(ctx, dsn)
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

	if app.db != nil {
		if sqlDB, err := app.db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("database shutdown error: %w", err))
			} else {
				slog.Info("Database pool closed cleanly")
			}
		}
	}

	if app.redisClient != nil {
		if err := app.redisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("redis client close error: %w", err))
		} else {
			slog.Info("Redis connection closed cleanly")
		}
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
