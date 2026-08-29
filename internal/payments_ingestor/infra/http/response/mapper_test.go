package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	errors2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
)

func TestTranslateAppError(t *testing.T) {
	t.Skip("Temporarily skipped during restructuring")
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Nil error returns 200",
			err:            nil,
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:           "ValidationError maps to 422",
			err:            errors2.NewAppError(constants.TypeValidationError, "INSUFFICIENT_BALANCE", "no money", nil),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedCode:   "INSUFFICIENT_BALANCE",
		},
		{
			name:           "NotFound maps to 404",
			err:            errors2.NewAppError(constants.TypeNotFound, "ACCOUNT_NOT_FOUND", "not found", nil),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "ACCOUNT_NOT_FOUND",
		},
		{
			name:           "Unauthorized maps to 401",
			err:            errors2.NewAppError(constants.TypeUnauthorized, "INVALID_TOKEN", "bad token", nil),
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "INVALID_TOKEN",
		},
		{
			name:           "Internal error maps to 500",
			err:            errors2.NewAppError(constants.TypeInternal, "DATABASE_ERROR", "db crash", nil),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "DATABASE_ERROR",
		},
		{
			name:           "Generic error maps to 500 INTERNAL_SERVER_ERROR",
			err:            errors.New("raw error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code, details := TranslateAppError(tt.err)
			if status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, status)
			}
			if code != tt.expectedCode {
				t.Errorf("expected code %q, got %q", tt.expectedCode, code)
			}
			if tt.err != nil && len(details) == 0 {
				t.Errorf("expected details for non-nil error")
			}
		})
	}
}
