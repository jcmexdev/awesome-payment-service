package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/app"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/config"
)

func main() {
	rootContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	application, err := app.NewApp(rootContext, cfg)
	if err != nil {
		panic(err)
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Http Server starting", "addr", application.Server.Addr)
		err := application.Server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		slog.Error("Failed to start HTTP server", "error", err)
		return
	case <-rootContext.Done():
		slog.Info("Shutdown signal received, starting graceful shutdown...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := application.Shutdown(shutdownCtx); err != nil {
			slog.Error("Graceful shutdown failed, forcing exit", "error", err)
			_ = application.Server.Close()
		} else {
			slog.Info("Server stopped cleanly")
		}
	}
}
