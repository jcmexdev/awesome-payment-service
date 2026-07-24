package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/domain/ports"
	"github.com/jcmexdev/payment-service/internal/infra/http/dtos"
	"github.com/jcmexdev/payment-service/internal/infra/http/response"
)

type PaymentsController interface {
	CreatePayment(w http.ResponseWriter, r *http.Request)
}

type PaymentsHandler struct {
	paymentUseCase ports.PaymentUseCase
}

func NewPaymentsHandler(paymentUseCase ports.PaymentUseCase) *PaymentsHandler {
	return &PaymentsHandler{paymentUseCase: paymentUseCase}
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

	// Invocar el caso de uso que encapsula la lógica de negocio y su span principal
	err = h.paymentUseCase.ProcessPayment(r.Context(), req.AccountID, req.Amount, refID)
	if err != nil {
		response.SendError(w, http.StatusUnprocessableEntity, "PAYMENT_FAILED", []response.ErrorDetail{
			{Reason: err.Error()},
		})
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
