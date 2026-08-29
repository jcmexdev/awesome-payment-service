package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	appConfig "github.com/jcmexdev/payment-service/internal/outbox_relayer/config"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/infrastructure"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/infrastructure/messaging"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/infrastructure/persistence/postgres"
	"github.com/jcmexdev/payment-service/pkg/telemetry"
)

func main() {
	ctx := context.Background()
	cfg := appConfig.LoadConfig()

	// Initialize OpenTelemetry if configured
	if cfg.OtelCollectorAddr != "" {
		shutdownTelemetry, err := telemetry.InitTracer(ctx, cfg.OtelServiceName, cfg.OtelCollectorAddr)
		if err != nil {
			slog.Error("Failed to initialize OpenTelemetry in Outbox Relayer", "error", err)
		} else {
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = shutdownTelemetry(shutdownCtx)
			}()
			slog.Info("OpenTelemetry initialized", "collector", cfg.OtelCollectorAddr, "service", cfg.OtelServiceName)
		}
	}

	endpointURL := os.Getenv("SQS_ENDPOINT") // http://floci:4566
	region := os.Getenv("AWS_REGION")        // us-east-1
	queueName := os.Getenv("SQS_QUEUE_NAME") // us-east-1

	// Resolver personalizado para redirigir AWS SQS hacia el contenedor de Floci
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("TEST", "TEST", "")))

	if err != nil {
		log.Fatalf("failed to load AWS configuration: %v", err)
	}

	slog.Info("Iniciando servicio Outbox Relayer",
		"env", cfg.Environment,
		"db_url", cfg.DBurl,
	)

	sqsClient := sqs.NewFromConfig(awsConfig, func(o *sqs.Options) {
		if endpointURL != "" {
			o.BaseEndpoint = aws.String(endpointURL) // Reemplazo oficial de AWS SDK v2
		}
	})

	log.Println("Cliente SQS listo usando BaseEndpoint (compatible con Floci)")
	_ = sqsClient

	sqsPublisher, err := messaging.NewSQSPublisher(ctx, sqsClient, queueName)
	if err != nil {
		log.Fatalf("failed to create sqs publisher: %v", err)
	}

	db, err := postgres.NewPostgresDB(cfg.DBurl)
	if err != nil {
		slog.Error("No se pudo conectar a la base de datos", "error", err)
		os.Exit(1)
	}

	outboxRepo := postgres.NewOutboxRepository(db)
	//publisher := messaging.NewStdoutPublisher()

	relayer := infrastructure.NewOutboxRelayer(
		outboxRepo,
		sqsPublisher,
		cfg.BatchSize,
		cfg.PollInterval,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go relayer.Start(ctx)

	slog.Info("Outbox Relayer en ejecución...")

	<-stop

	cancel()

	time.Sleep(1 * time.Second)
	slog.Info("Proceso Outbox Relayer finalizado de forma limpia.")
}
