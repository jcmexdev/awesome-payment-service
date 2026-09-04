package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/jcmexdev/payment-service/internal/events_consumer/application"
	"github.com/jcmexdev/payment-service/internal/events_consumer/config"
	"github.com/jcmexdev/payment-service/internal/events_consumer/infrastructure/messaging"
	"github.com/jcmexdev/payment-service/internal/events_consumer/infrastructure/persistence/postgres"
)

func main() {
	// Escuchar señales del SO para apagado limpio (SIGINT / SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.LoadConfig()

	slog.Info("Iniciando servicio EventConsumer",
		"env", cfg.Environment,
		"db_url", cfg.DBurl,
	)

	// 1. Inicializar OpenTelemetry si está configurada la dirección del collector
	var tracer trace.Tracer = otel.Tracer("events_consumer_noop") // Fallback NoOp
	if cfg.OtelCollectorAddr != "" {
		tp, shutdownTelemetry, err := initTracer(ctx, cfg.OtelServiceName, cfg.OtelCollectorAddr)
		if err != nil {
			slog.Error("Fallo al inicializar OpenTelemetry", "error", err)
		} else {
			// Flush de trazas acumuladas en memoria antes de apagar la aplicación
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := shutdownTelemetry(shutdownCtx); err != nil {
					slog.Error("Error cerrando OpenTelemetry", "error", err)
				}
			}()
			slog.Info("OpenTelemetry inicializado", "collector", cfg.OtelCollectorAddr, "service", cfg.OtelServiceName)
			tracer = tp.Tracer("events_consumer/sqs-worker")
		}
	}

	sqsClient, err := getSQSClient(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	db, err := postgres.NewPostgresDB(cfg.DBurl)
	if err != nil {
		log.Fatal(err)
	}

	paymentRepo := postgres.NewPaymentsRepository(db)
	paymentProcessor := application.NewPaymentProcessor(paymentRepo)

	// 2. Inyectar la URL de la cola y la instancia real de tracer
	worker := messaging.NewSQSWorker(
		sqsClient,
		cfg.SQSUrl, //  👈 Instancia válida de trace.Tracer
		paymentProcessor,
		tracer,
	)

	// 3. Arrancar el consumo con nivel de concurrencia deseado
	slog.Info("Arrancando el SQS Worker...")
	if err := worker.Start(ctx, 5); err != nil {
		slog.Error("El SQS Worker terminó con un error", "error", err)
	}
}

// initTracer configura el exporter OTLP y el TracerProvider de OpenTelemetry
func initTracer(ctx context.Context, serviceName, collectorURL string) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(collectorURL),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("falló al crear el exportador OTLP: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("falló al crear el recurso de OTel: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	// Configurar el propagador estándar W3C para reconocer la cabecera traceparent
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, tp.Shutdown, nil
}

func getSQSClient(ctx context.Context, cfg *config.Config) (*sqs.Client, error) {
	defaultConfig, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(cfg.SQSRegion),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("TEST", "TEST", "")))

	if err != nil {
		return nil, err
	}
	return sqs.NewFromConfig(defaultConfig, func(o *sqs.Options) {
		if cfg.SQSEndpoint != "" {
			o.BaseEndpoint = aws.String(cfg.SQSEndpoint)
		}
	}), nil
}
