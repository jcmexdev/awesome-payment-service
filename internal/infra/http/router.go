package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jcmexdev/payment-service/internal/infra/http/handler"
)

func NewRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/ping", handler.HealthCheck)
	router.Post("/v1/payments", handler.CreatePaymentHandler)
	return router
}
