package telemetry

import (
	"context"
	"log/slog"

	"github.com/jcmexdev/payment-service/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

type ContextHandler struct {
	slog.Handler
}

func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: h}
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx == nil {
		return h.Handler.Handle(ctx, r)
	}

	// Inyectar service_name al nivel raíz
	r.AddAttrs(slog.String("service_name", "payment-service"))

	// Extraer request_id del contexto
	if reqID, ok := ctx.Value(domain.ContextKeyRequestID).(string); ok && reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}

	// Extraer trace_id de OpenTelemetry o alternativamente del contexto manual
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		r.AddAttrs(slog.String("trace_id", spanCtx.TraceID().String()))
	} else {
		if traceID, ok := ctx.Value(domain.ContextKeyTraceID).(string); ok && traceID != "" {
			r.AddAttrs(slog.String("trace_id", traceID))
		}
	}

	return h.Handler.Handle(ctx, r)
}
