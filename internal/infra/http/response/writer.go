package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jcmexdev/payment-service/internal/domain/constants"
	appErrors "github.com/jcmexdev/payment-service/internal/domain/errors"
)

// HandleError centraliza el logging estructurado del error y envía la respuesta HTTP mapeada al cliente
func HandleError(w http.ResponseWriter, r *http.Request, err error, logMsg string) {
	if err == nil {
		return
	}

	ctx := r.Context()
	var appErr *appErrors.AppError

	if errors.As(err, &appErr) {
		logArgs := []any{
			slog.String("code", appErr.Code),
			slog.String("message", appErr.Message),
		}
		if appErr.Err != nil {
			logArgs = append(logArgs, slog.String("error", appErr.Err.Error()))
		}
		// Agrupación híbrida en error_details
		if len(appErr.Context) > 0 {
			var groupAttrs []any
			for k, v := range appErr.Context {
				groupAttrs = append(groupAttrs, slog.Any(k, v))
			}
			logArgs = append(logArgs, slog.Group("error_details", groupAttrs...))
		}
		slog.ErrorContext(ctx, logMsg, logArgs...)
	} else {
		slog.ErrorContext(ctx, logMsg, "error", err)
	}

	// Mapear y enviar error por HTTP
	status, code, details := TranslateAppError(err)
	SendError(w, r, status, code, details)
}

func SendError(w http.ResponseWriter, r *http.Request, statusCode int, errorCode string, details []ErrorDetail) {
	var reqID, traceID string
	if r != nil {
		ctx := r.Context()
		reqID, _ = ctx.Value(constants.ContextKeyRequestID).(string)
		traceID, _ = ctx.Value(constants.ContextKeyTraceID).(string)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := ErrorResponse{
		Status:  "error",
		Code:    errorCode,
		Message: GetMessage(errorCode),
		Meta: &Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: reqID,
			TraceID:   traceID,
			Version:   "v1",
		},
		Details: details,
	}
	json.NewEncoder(w).Encode(resp)
}

func SendSuccess[T any](w http.ResponseWriter, r *http.Request, statusCode int, successCode string, data T) {
	var reqID, traceID string
	if r != nil {
		ctx := r.Context()
		reqID, _ = ctx.Value(constants.ContextKeyRequestID).(string)
		traceID, _ = ctx.Value(constants.ContextKeyTraceID).(string)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := SuccessResponse[T]{
		Status:  "success",
		Code:    successCode,
		Message: GetMessage(successCode),
		Meta: &Meta{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: reqID,
			TraceID:   traceID,
			Version:   "v1",
		},
		Data: data,
	}

	json.NewEncoder(w).Encode(resp)
}
