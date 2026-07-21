package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jcmexdev/payment-service/internal/domain/ports"
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

		acquired, record, err := m.repo.Lock(r.Context(), key, m.ttl)
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
			if err := m.repo.Save(r.Context(), key, wrapper.statusCode, wrapper.body.Bytes(), m.ttl); err != nil {
				slog.Error("idempotency_middleware_save_error", "key", key, "error", err)
			}
		}
	})
}
