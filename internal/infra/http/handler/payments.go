package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/constants"
	"github.com/jcmexdev/payment-service/internal/domain/ports"
	"github.com/jcmexdev/payment-service/internal/infra/http/consts"
	"github.com/jcmexdev/payment-service/internal/infra/http/dtos"
	"github.com/jcmexdev/payment-service/internal/infra/http/response"
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
		response.SendError(w, r, http.StatusBadRequest, response.CodeMalformedJSON, nil)
		return
	}

	if req.UserID == "" {
		response.SendError(w, r, http.StatusBadRequest, response.CodeMissingUserId, []response.ErrorDetail{
			{Field: "user_id", Reason: "user_id is required"},
		})
		return
	}
	if len(req.Currency) != 3 {
		response.SendError(w, r, http.StatusBadRequest, "INVALID_CURRENCY", []response.ErrorDetail{
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
	if err != nil {
		response.HandleError(w, r, err, "CreateAccount failed")
		return
	}

	slog.InfoContext(ctx, "CreateAccount successful", "account_id", account.ID, "user_id", account.UserID)
	response.SendSuccess(w, r, http.StatusCreated, response.CodeAccountCreated, dtos.NewCreateAccountResponse(&account))
}

func (h PaymentsHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req dtos.PaymentRequestDTO
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.SendError(w, r, http.StatusBadRequest, response.CodeMalformedJSON, nil)
		return
	}

	if req.AccountID == "" {
		response.SendError(w, r, http.StatusBadRequest, "MISSING_ACCOUNT_ID", []response.ErrorDetail{
			{Field: "account_id", Reason: "account_id is required"},
		})
		return
	}
	if req.Amount <= 0 {
		response.SendError(w, r, http.StatusBadRequest, "INVALID_AMOUNT", []response.ErrorDetail{
			{Field: "amount", Reason: "amount must be greater than zero"},
		})
		return
	}

	refID := r.Header.Get(consts.HeaderRequestID)
	if refID == "" {
		refID = uuid.New().String()
	}

	// Capturar cabeceras de Chaos Engineering y agregarlas al contexto
	ctx := r.Context()
	if delayVal := r.Header.Get("X-Simulate-Delay"); delayVal != "" {
		if d, err := time.ParseDuration(delayVal); err == nil {
			ctx = context.WithValue(ctx, constants.ContextKeySimulateDelay, d)
		}
	}
	if failVal := r.Header.Get("X-Simulate-Error"); failVal == "true" {
		ctx = context.WithValue(ctx, constants.ContextKeySimulateError, true)
	}

	// Invocar el caso de uso que encapsula la lógica de negocio y su span principal
	err = h.paymentUseCase.ProcessPayment(ctx, req.AccountID, req.Amount, refID)
	if err != nil {
		response.HandleError(w, r, err, "ProcessPayment failed")
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
	slog.InfoContext(ctx, "ProcessPayment successful", "account_id", req.AccountID, "reference_id", refID)
	response.SendSuccess(w, r, http.StatusAccepted, response.CodePaymentAccepted, dto)
}
