package main

import (
	"net/http"

	router "github.com/jcmexdev/payment-service/internal/infra/http"
	"github.com/jcmexdev/payment-service/internal/infra/http/handler"
)

func main() {
	r := router.NewRouter(
		router.WithHealthController(handler.NewHealthHandler()),
		router.WithPaymentsController(handler.NewPaymentsHandler()),
	)
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		return
	}
}
