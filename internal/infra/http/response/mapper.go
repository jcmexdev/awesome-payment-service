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
		case appErrors.TypeConflict:
			status = http.StatusConflict

		default:
			status = http.StatusInternalServerError
		}

		var details []ErrorDetail

		for field, reasonVal := range appErr.Details {
			if reasonStr, ok := reasonVal.(string); ok {
				details = append(details, ErrorDetail{
					Field:  field,
					Reason: reasonStr,
				})
			}
		}

		return status, appErr.Code, details
	}

	// Fallback for generic, unhandled errors
	return http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", []ErrorDetail{
		{Reason: err.Error()},
	}
}
