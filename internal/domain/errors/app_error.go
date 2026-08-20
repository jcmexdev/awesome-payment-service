package errors

import "fmt"

type ErrorType string

const (
	TypeNotFound        ErrorType = "NOT_FOUND"
	TypeValidationError ErrorType = "VALIDATION"
	TypeUnauthorized    ErrorType = "UNAUTHORIZED"
	TypeInternal        ErrorType = "INTERNAL"
)

type AppError struct {
	Type    ErrorType
	Code    string
	Message string
	Err     error
	Context map[string]any // Metadatos clave-valor adicionales para logs estructurados
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

// WithContext permite adjuntar información estructurada al error
func (e *AppError) WithContext(key string, value any) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

func NewAppError(errType ErrorType, code string, msg string, original error) *AppError {
	return &AppError{
		Type:    errType,
		Code:    code,
		Message: msg,
		Err:     original,
	}
}
