package handler

import (
	"net/http"

	"github.com/jcmexdev/payment-service/internal/infra/http/response"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {

	response.SendSuccess(w, http.StatusOK, response.CodeSystemHealthy, struct{}{})
}
