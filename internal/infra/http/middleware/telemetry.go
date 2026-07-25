package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

const (
	HeaderRequestID = "X-Request-ID"
	HeaderTraceID   = "X-Trace-ID"
)

func TelemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Obtener o generar Request ID
		reqID := r.Header.Get(HeaderRequestID)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// 2. Obtener o generar Trace ID
		var traceID string
		spanCtx := trace.SpanContextFromContext(ctx)
		if spanCtx.IsValid() {
			traceID = spanCtx.TraceID().String()
		} else {
			traceID = r.Header.Get(HeaderTraceID)
			if traceID == "" {
				// Generar ID de 16 bytes (32 caracteres hexadecimales) compatible con OTEL
				bytes := make([]byte, 16)
				if _, err := rand.Read(bytes); err == nil {
					traceID = hex.EncodeToString(bytes)
				} else {
					traceID = uuid.New().String()
				}
			}
		}

		// 3. Inyectar IDs en el Contexto
		ctx = context.WithValue(ctx, domain.ContextKeyRequestID, reqID)
		ctx = context.WithValue(ctx, domain.ContextKeyTraceID, traceID)

		// 4. Inyectar encabezados en la respuesta HTTP
		w.Header().Set(HeaderRequestID, reqID)
		w.Header().Set(HeaderTraceID, traceID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
