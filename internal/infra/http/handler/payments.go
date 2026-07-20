package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jcmexdev/payment-service/internal/domain"
	"github.com/jcmexdev/payment-service/internal/infra/http/dtos"
	"github.com/jcmexdev/payment-service/internal/infra/http/response"
)

func CreatePaymentHandler(w http.ResponseWriter, r *http.Request) {
	var req dtos.PaymentRequestDTO
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.SendError(w, http.StatusBadRequest, response.CodeMalformedJSON, nil)
		return
	}

	var payment domain.Payment
	dto := dtos.PaymentFromDomain(&payment)
	response.SendSuccess(w, http.StatusAccepted, response.CodePaymentAccepted, dto)
}
