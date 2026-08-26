package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/consts"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

func TelemetryMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			reqID := r.Header.Get(consts.HeaderRequestID)
			if reqID == "" {
				reqID = uuid.New().String()
			}

			propagator := otel.GetTextMapPropagator()
			ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))

			spanName := r.Method + " " + r.URL.Path
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.URLPath(r.URL.Path),
					attribute.String("request.id", reqID),
				),
			)
			defer span.End()

			spanCtx := trace.SpanContextFromContext(ctx)
			var traceID string
			if spanCtx.IsValid() {
				traceID = spanCtx.TraceID().String()
			}

			ctx = context.WithValue(ctx, constants.ContextKeyRequestID, reqID)
			if traceID != "" {
				ctx = context.WithValue(ctx, constants.ContextKeyTraceID, traceID)
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			ww.Header().Set(consts.HeaderRequestID, reqID)
			if traceID != "" {
				ww.Header().Set(consts.HeaderTraceID, traceID)
			}

			next.ServeHTTP(ww, r.WithContext(ctx))

			span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(ww.Status()))

			if ww.Status() >= 500 {
				span.SetStatus(codes.Error, "Internal Server Error")
			}
		})
	}
}
