package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jcmexdev/payment-service/internal/domain"
	appmiddleware "github.com/jcmexdev/payment-service/internal/infra/http/middleware"
)

type mockIdempotencyRepo struct {
	locked bool
	saved  bool
}

func (m *mockIdempotencyRepo) Lock(ctx context.Context, key string, ttl time.Duration) (bool, *domain.IdempotencyRecord, error) {
	m.locked = true
	return true, nil, nil
}

func (m *mockIdempotencyRepo) Save(ctx context.Context, key string, responseCode int, responseBody []byte, ttl time.Duration) error {
	m.saved = true
	return nil
}

type mockPaymentsController struct{}

func (m *mockPaymentsController) CreatePayment(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"status":"created"}`))
}

func (m *mockPaymentsController) CreateAccount(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"status":"account_created"}`))
}

type mockHealthController struct{}

func (m *mockHealthController) Health(w http.ResponseWriter, r *http.Request) {}

func TestRouterIdempotency(t *testing.T) {
	repo := &mockIdempotencyRepo{}
	middlewareInstance := appmiddleware.NewIdempotencyMiddleware(repo, time.Minute)
	routerInstance := NewRouter(
		WithHealthController(&mockHealthController{}),
		WithPaymentsController(&mockPaymentsController{}),
		WithIdempotencyMiddleware(middlewareInstance),
	)

	// Case 1: GET to /v1/payment (should NOT lock because method is GET)
	req := httptest.NewRequest(http.MethodGet, "/v1/payment", nil)
	req.Header.Set("Idempotency-Key", "test-key-1")
	rec := httptest.NewRecorder()
	routerInstance.ServeHTTP(rec, req)

	if repo.locked {
		t.Error("Expected GET request not to invoke lock in idempotency middleware")
	}

	// Case 2: POST to /v1/payment (singular - unregistered route)
	repo.locked = false
	repo.saved = false
	req = httptest.NewRequest(http.MethodPost, "/v1/payment", nil)
	req.Header.Set("Idempotency-Key", "test-key-2")
	rec = httptest.NewRecorder()
	routerInstance.ServeHTTP(rec, req)

	if !repo.locked {
		t.Error("Expected POST request to /v1/payment to invoke lock in idempotency middleware")
	}
	if !repo.saved {
		t.Error("Expected idempotency middleware to save the response for unregistered route POST /v1/payment")
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status code 404, got %d", rec.Code)
	}

	// Case 3: POST to /v1/payments (plural - registered route)
	repo.locked = false
	repo.saved = false
	req = httptest.NewRequest(http.MethodPost, "/v1/payments", nil)
	req.Header.Set("Idempotency-Key", "test-key-3")
	rec = httptest.NewRecorder()
	routerInstance.ServeHTTP(rec, req)

	if !repo.locked {
		t.Error("Expected POST request to /v1/payments to invoke lock in idempotency middleware")
	}
	if !repo.saved {
		t.Error("Expected idempotency middleware to save the response for registered route POST /v1/payments")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", rec.Code)
	}
}
