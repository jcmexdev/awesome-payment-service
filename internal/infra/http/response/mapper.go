package response

import (
	"errors"
	"net/http"

	appErrors "github.com/jcmexdev/payment-service/internal/domain/errors"
)

// TranslateAppError translates an appErrors.AppError into HTTP status code, API error code, and error details
func TranslateAppError(err error) (int, string, []ErrorDetail) {
	if err == nil {
		return http.StatusOK, "", nil
	}

	var appErr *appErrors.AppError
	if errors.As(err, &appErr) {
		var status int
		switch appErr.Type {
		case appErrors.TypeNotFound:
			status = http.StatusNotFound
		case appErrors.TypeValidationError:
			status = http.StatusUnprocessableEntity
		case appErrors.TypeUnauthorized:
			status = http.StatusUnauthorized
		case appErrors.TypeInternal:
			status = http.StatusInternalServerError
		default:
			status = http.StatusInternalServerError
		}

		details := []ErrorDetail{
			{Reason: appErr.Message},
		}

		return status, appErr.Code, details
	}

	// Fallback for generic, unhandled errors
	return http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", []ErrorDetail{
		{Reason: err.Error()},
	}
}
