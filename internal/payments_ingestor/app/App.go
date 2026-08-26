package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	usecase2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/app/usecase"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/config"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/cache"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/cache/gorm"
	rediscache "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/cache/redis"
	router "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http"
	handler2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/handler"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/middleware"
	postgres2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/persistence/postgres"
	telemetry2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/telemetry"
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
	var level slog.Level
	switch cfg.LogLevel {
	case "debug", "DEBUG":
		level = slog.LevelDebug
	case "warn", "WARN":
		level = slog.LevelWarn
	case "error", "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(telemetry2.NewContextHandler(jsonHandler))
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
		shutdownTelemetry, err = telemetry2.InitTracer(ctx, cfg.OtelServiceName, cfg.OtelCollectorAddr)
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

	accountRepository := postgres2.NewAccountRepository(gormDB)
	paymentsRepository := postgres2.NewPaymentsRepository(gormDB)
	outboxRepository := postgres2.NewOutboxRepository(gormDB)
	unitOfWork := postgres2.NewGormUnitOfWork(gormDB)

	accountUseCase := usecase2.NewCreateAccountUseCase(accountRepository)
	paymentUseCase := usecase2.NewAuthorizePaymentUseCase(paymentsRepository, outboxRepository, unitOfWork)

	r := router.NewRouter(
		router.WithHealthController(handler2.NewHealthHandler()),
		router.WithPaymentsController(handler2.NewPaymentsHandler(paymentUseCase)),
		router.WithIdempotencyMiddleware(idempMiddleware),
		router.WithAccountController(handler2.NewAccountHandler(accountUseCase, cfg.ServiceName)),
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

	client.AddHook(telemetry2.NewOpenTelemetryRedisHook())
	client.AddHook(telemetry2.NewPrometheusRedisHook())

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
