package errors

import (
	"fmt"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
)

type AppError struct {
	Type       constants.ResponseType
	Code       string
	Message    string
	Err        error
	LogContext map[string]any
	Details    map[string]any
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s (Internal: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// WithLogContext permite adjuntar información estructurada al error
func (e *AppError) WithLogContext(key string, value any) *AppError {
	if e.LogContext == nil {
		e.LogContext = make(map[string]any)
	}
	e.LogContext[key] = value
	return e
}

func (e *AppError) WithDetail(key string, value any) {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
}

func NewAppError(errType constants.ResponseType, code string, msg string, original error) *AppError {
	return &AppError{
		Type:    errType,
		Code:    code,
		Message: msg,
		Err:     original,
	}
}
