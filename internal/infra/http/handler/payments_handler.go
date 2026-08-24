package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/domain/constants"
	"github.com/jcmexdev/payment-service/internal/domain/errors"
	"github.com/jcmexdev/payment-service/internal/domain/ports"
	"github.com/jcmexdev/payment-service/internal/infra/http/consts"
	"github.com/jcmexdev/payment-service/internal/infra/http/dtos"
	"github.com/jcmexdev/payment-service/internal/infra/http/response"
)

type PaymentsController interface {
	CreatePayment(w http.ResponseWriter, r *http.Request)
}

type PaymentsHandler struct {
	paymentUseCase ports.AuthorizePaymentUseCase
}

func NewPaymentsHandler(paymentUseCase ports.AuthorizePaymentUseCase) *PaymentsHandler {
	return &PaymentsHandler{paymentUseCase: paymentUseCase}
}

func (h PaymentsHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req dtos.PaymentRequestDTO
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.SendError(w, r, http.StatusBadRequest, errors.CodeMalformedJSON, nil)
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
	/*_,err = h.paymentUseCase.Execute(ctx, req.AccountID, req.Amount, refID)
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
	response.SendSuccess(w, r, http.StatusAccepted, errors.CodePaymentAccepted, dto)*/
}
