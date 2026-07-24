package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jcmexdev/payment-service/internal/domain/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const HeaderIdempotencyKey = "Idempotency-Key"

type IdempotencyMiddleware struct {
	repo ports.IdempotencyRepository
	ttl  time.Duration
}

func NewIdempotencyMiddleware(repo ports.IdempotencyRepository, ttl time.Duration) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{
		repo: repo,
		ttl:  ttl,
	}
}

func (m *IdempotencyMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get(HeaderIdempotencyKey)
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		tr := otel.Tracer("idempotency-middleware")

		// 1. Wrap the Lock and read check phase in a child span
		ctx, checkSpan := tr.Start(r.Context(), "idempotency.check", trace.WithAttributes(
			attribute.String("idempotency.key", key),
		))
		acquired, record, err := m.repo.Lock(ctx, key, m.ttl)
		if err != nil {
			checkSpan.RecordError(err)
			checkSpan.SetStatus(codes.Error, err.Error())
		} else {
			checkSpan.SetStatus(codes.Ok, "idempotency check completed")
		}
		checkSpan.End()

		if err != nil {
			slog.Error("idempotency_middleware_lock_error", "key", key, "error", err)
			next.ServeHTTP(w, r)
			return
		}

		if !acquired && record != nil {
			if record.IsProcessing() {
				http.Error(w, `{"error":"request_in_progress","message":"A request with this idempotency key is currently processing"}`, http.StatusConflict)
				return
			}

			if record.IsCompleted() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT-IDEMPOTENCY")
				w.WriteHeader(record.ResponseCode)
				_, _ = w.Write(record.ResponseBody)
				return
			}
		}

		wrapper := newResponseWriterWrapper(w)

		next.ServeHTTP(wrapper, r)

		if wrapper.statusCode < http.StatusInternalServerError {
			// 2. Wrap the Save phase in a child span
			ctx, saveSpan := tr.Start(r.Context(), "idempotency.save", trace.WithAttributes(
				attribute.String("idempotency.key", key),
				attribute.Int("http.status_code", wrapper.statusCode),
			))
			if err := m.repo.Save(ctx, key, wrapper.statusCode, wrapper.body.Bytes(), m.ttl); err != nil {
				saveSpan.RecordError(err)
				saveSpan.SetStatus(codes.Error, err.Error())
				slog.Error("idempotency_middleware_save_error", "key", key, "error", err)
			} else {
				saveSpan.SetStatus(codes.Ok, "idempotency response saved")
			}
			saveSpan.End()
		}
	})
}
