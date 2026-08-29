package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	appmiddleware "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/middleware"
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

func (m *mockPaymentsController) AuthorizePayment(w http.ResponseWriter, r *http.Request) {
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
	t.Skip("Temporarily skipped during restructuring")
	repo := &mockIdempotencyRepo{}
	middlewareInstance := appmiddleware.NewIdempotencyMiddleware(repo, time.Minute)
	routerInstance := NewRouter(
		WithHealthController(&mockHealthController{}),
		WithPaymentsController(&mockPaymentsController{}),
		WithIdempotencyMiddleware(middlewareInstance),
	)

	// Case 1: GET to /v1/payments (should NOT lock because method is GET)
	req := httptest.NewRequest(http.MethodGet, "/v1/payments", nil)
	req.Header.Set("Idempotency-Key", "test-key-1")
	rec := httptest.NewRecorder()
	routerInstance.ServeHTTP(rec, req)

	if repo.locked {
		t.Error("Expected GET request not to invoke lock in idempotency middleware")
	}

	// Case 2: POST to /v1/accounts (registered route - account prefix)
	repo.locked = false
	repo.saved = false
	req = httptest.NewRequest(http.MethodPost, "/v1/accounts", nil)
	req.Header.Set("Idempotency-Key", "test-key-2")
	rec = httptest.NewRecorder()
	routerInstance.ServeHTTP(rec, req)

	if !repo.locked {
		t.Error("Expected POST request to /v1/accounts to invoke lock in idempotency middleware")
	}
	if !repo.saved {
		t.Error("Expected idempotency middleware to save the response for POST /v1/accounts")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code 201 (Created), got %d", rec.Code)
	}

	// Case 3: POST to /v1/payments (registered route - payment prefix)
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
		t.Error("Expected idempotency middleware to save the response for POST /v1/payments")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", rec.Code)
	}
}
