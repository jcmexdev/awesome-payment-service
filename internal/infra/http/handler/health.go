package handler

import (
	"net/http"

	"github.com/jcmexdev/payment-service/internal/domain/errors"
	"github.com/jcmexdev/payment-service/internal/infra/http/response"
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
	response.SendSuccess(w, r, http.StatusOK, errors.CodeSystemHealthy, struct{}{})
}
