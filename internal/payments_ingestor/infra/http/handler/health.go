package handler

import (
	"net/http"

	"github.com/jcmexdev/payment-service/internal/payments_ingestor/domain/constants"
	"github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/response"
)

type HealthCheckController interface {
	Health(w http.ResponseWriter, r *http.Request)
}

type HealthCheckHandler struct {
}

func NewHealthHandler() *HealthCheckHandler {
	return &HealthCheckHandler{}
}

func (h HealthCheckHandler) Health(w http.ResponseWriter, r *http.Request) {
	response.SendSuccess(w, r, http.StatusOK, constants.CodeSystemHealthy, struct{}{})
}
