package response

import (
	"errors"
	"net/http"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	appError "github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/errors"
)

// TranslateAppError translates an appErrorors.appErroror into HTTP status code, API error code, and error details
func TranslateAppError(err error) (int, string, []ErrorDetail) {
	if err == nil {
		return http.StatusOK, "", nil
	}

	var appErr *appError.AppError
	if errors.As(err, &appErr) {
		var status int
		switch appErr.Type {
		case constants.TypeNotFound:
			status = http.StatusNotFound
		case constants.TypeValidationError:
			status = http.StatusUnprocessableEntity
		case constants.TypeUnauthorized:
			status = http.StatusUnauthorized
		case constants.TypeInternal:
			status = http.StatusInternalServerError
		case constants.TypeConflict:
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
