package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/ports"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/consts"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/dtos"
	response2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/response"
)

type PaymentsController interface {
	AuthorizePayment(w http.ResponseWriter, r *http.Request)
}

type PaymentsHandler struct {
	paymentUseCase ports.AuthorizePaymentUseCase
}

func NewPaymentsHandler(paymentUseCase ports.AuthorizePaymentUseCase) *PaymentsHandler {
	return &PaymentsHandler{paymentUseCase: paymentUseCase}
}

func (h PaymentsHandler) AuthorizePayment(w http.ResponseWriter, r *http.Request) {
	var req domain.AuthorizePaymentRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response2.SendError(w, r, http.StatusBadRequest, constants.CodeMalformedJSON, nil)
		return
	}

	if req.AccountID == "" {
		response2.SendError(w, r, http.StatusBadRequest, "MISSING_ACCOUNT_ID", []response2.ErrorDetail{
			{Field: "account_id", Reason: "account_id is required"},
		})
		return
	}
	if req.Amount <= 0 {
		response2.SendError(w, r, http.StatusBadRequest, "INVALID_AMOUNT", []response2.ErrorDetail{
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

	payment, err := h.paymentUseCase.Execute(ctx, &req)
	if err != nil {
		response2.HandleError(w, r, err, "ProcessPayment failed")
		return
	}

	/*payment := domain.Payment{
		ID:        refID,
		AccountID: req.AccountID,
		Amount:    req.Amount,
		Currency:  req.Currency,
		Status:    "SETTLED",
		CreatedAt: time.Now().UTC(),
	}*/

	dto := dtos.PaymentFromDomain(payment)
	//slog.InfoContext(ctx, "ProcessPayment successful", "account_id", req.AccountID, "reference_id", refID)
	response2.SendSuccess(w, r, http.StatusAccepted, constants.CodePaymentAccepted, dto)
}
