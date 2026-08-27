package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jcmexdev/payment-service/internal/outbox_relayer/config"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/infrastructure"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/infrastructure/messaging"
	"github.com/jcmexdev/payment-service/internal/outbox_relayer/infrastructure/persistence/postgres"
)

func main() {
	cfg := config.LoadConfig()

	slog.Info("Iniciando servicio Outbox Relayer",
		"env", cfg.Environment,
		"db_url", cfg.DBurl,
	)

	db, err := postgres.NewPostgresDB(cfg.DBurl)
	if err != nil {
		slog.Error("No se pudo conectar a la base de datos", "error", err)
		os.Exit(1)
	}

	outboxRepo := postgres.NewOutboxRepository(db)
	publisher := messaging.NewStdoutPublisher()

	relayer := infrastructure.NewOutboxRelayer(
		outboxRepo,
		publisher,
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
