package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/domain"
	appErrors "github.com/jcmexdev/payment-service/internal/domain/errors"
	"github.com/jcmexdev/payment-service/internal/domain/ports"
	"github.com/jcmexdev/payment-service/internal/infra/http/dtos"
	"github.com/jcmexdev/payment-service/internal/infra/http/response"
	"go.opentelemetry.io/otel/trace"
)

type PaymentsController interface {
	CreatePayment(w http.ResponseWriter, r *http.Request)
	CreateAccount(w http.ResponseWriter, r *http.Request)
}

type PaymentsHandler struct {
	paymentUseCase ports.PaymentUseCase
}

func NewPaymentsHandler(paymentUseCase ports.PaymentUseCase) *PaymentsHandler {
	return &PaymentsHandler{paymentUseCase: paymentUseCase}
}

func (h PaymentsHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		Currency string `json:"currency"`
		Balance  int64  `json:"balance"` // En centavos
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.SendError(w, http.StatusBadRequest, response.CodeMalformedJSON, nil)
		return
	}

	if req.UserID == "" {
		response.SendError(w, http.StatusBadRequest, "MISSING_USER_ID", []response.ErrorDetail{
			{Field: "user_id", Reason: "user_id is required"},
		})
		return
	}
	if len(req.Currency) != 3 {
		response.SendError(w, http.StatusBadRequest, "INVALID_CURRENCY", []response.ErrorDetail{
			{Field: "currency", Reason: "currency must be a 3-character ISO-4217 code"},
		})
		return
	}

	account := domain.Account{
		ID:            uuid.New().String(),
		UserID:        req.UserID,
		Currency:      req.Currency,
		CachedBalance: req.Balance,
		Version:       0,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	ctx := r.Context()
	err = h.paymentUseCase.CreateAccount(ctx, &account)
	
	// Escribir el TraceID de OpenTelemetry en los headers de respuesta si está activo
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		w.Header().Set("X-Trace-ID", spanCtx.TraceID().String())
	}

	if err != nil {
		var appErr *appErrors.AppError
		if errors.As(err, &appErr) {
			logArgs := []any{
				slog.String("code", appErr.Code),
				slog.String("message", appErr.Message),
			}
			if appErr.Err != nil {
				logArgs = append(logArgs, slog.String("error", appErr.Err.Error()))
			}
			for k, v := range appErr.Context {
				logArgs = append(logArgs, slog.Any(k, v))
			}
			slog.Error("CreateAccount failed", logArgs...)
		} else {
			slog.Error("CreateAccount unexpected error", "error", err)
		}

		status, code, details := response.TranslateAppError(err)
		response.SendError(w, status, code, details)
		return
	}

	response.SendSuccess(w, http.StatusCreated, "ACCOUNT_CREATED", account)
}

func (h PaymentsHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req dtos.PaymentRequestDTO
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.SendError(w, http.StatusBadRequest, response.CodeMalformedJSON, nil)
		return
	}

	if req.AccountID == "" {
		response.SendError(w, http.StatusBadRequest, "MISSING_ACCOUNT_ID", []response.ErrorDetail{
			{Field: "account_id", Reason: "account_id is required"},
		})
		return
	}
	if req.Amount <= 0 {
		response.SendError(w, http.StatusBadRequest, "INVALID_AMOUNT", []response.ErrorDetail{
			{Field: "amount", Reason: "amount must be greater than zero"},
		})
		return
	}

	refID := r.Header.Get("Idempotency-Key")
	if refID == "" {
		refID = uuid.New().String()
	}

	// Capturar cabeceras de Chaos Engineering y agregarlas al contexto
	ctx := r.Context()
	if delayVal := r.Header.Get("X-Simulate-Delay"); delayVal != "" {
		if d, err := time.ParseDuration(delayVal); err == nil {
			ctx = context.WithValue(ctx, domain.ContextKeySimulateDelay, d)
		}
	}
	if failVal := r.Header.Get("X-Simulate-Error"); failVal == "true" {
		ctx = context.WithValue(ctx, domain.ContextKeySimulateError, true)
	}

	// Invocar el caso de uso que encapsula la lógica de negocio y su span principal
	err = h.paymentUseCase.ProcessPayment(ctx, req.AccountID, req.Amount, refID)
	
	// Escribir el TraceID de OpenTelemetry en los headers de respuesta si está activo
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		w.Header().Set("X-Trace-ID", spanCtx.TraceID().String())
	}

	if err != nil {
		var appErr *appErrors.AppError
		if errors.As(err, &appErr) {
			logArgs := []any{
				slog.String("code", appErr.Code),
				slog.String("message", appErr.Message),
			}
			if appErr.Err != nil {
				logArgs = append(logArgs, slog.String("error", appErr.Err.Error()))
			}
			for k, v := range appErr.Context {
				logArgs = append(logArgs, slog.Any(k, v))
			}
			slog.Error("ProcessPayment failed", logArgs...)
		} else {
			slog.Error("ProcessPayment unexpected error", "error", err)
		}

		status, code, details := response.TranslateAppError(err)
		response.SendError(w, status, code, details)
		return
	}

	payment := domain.Payment{
		ID:        refID,
		AccountID: req.AccountID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Status:    "SETTLED",
		CreatedAt: time.Now().UTC(),
	}

	dto := dtos.PaymentFromDomain(&payment)
	response.SendSuccess(w, http.StatusAccepted, response.CodePaymentAccepted, dto)
}
