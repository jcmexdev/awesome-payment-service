package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jcmexdev/payment-service/internal/infra/http/handler"
	appmiddleware "github.com/jcmexdev/payment-service/internal/infra/http/middleware"
	"github.com/riandyrn/otelchi"
)

type router struct {
	healthController      handler.HealthCheckController
	paymentsController    handler.PaymentsController
	idempotencyMiddleware *appmiddleware.IdempotencyMiddleware
}

type Options func(r *router)

func WithIdempotencyMiddleware(idempotencyMiddleware *appmiddleware.IdempotencyMiddleware) Options {
	return func(r *router) {
		r.idempotencyMiddleware = idempotencyMiddleware
	}
}

func WithHealthController(checkHandler handler.HealthCheckController) Options {
	return func(r *router) {
		r.healthController = checkHandler
	}
}

func WithPaymentsController(checkHandler handler.PaymentsController) Options {
	return func(r *router) {
		r.paymentsController = checkHandler
	}
}

func NewRouter(options ...Options) *chi.Mux {
	r := &router{}
	mux := chi.NewRouter()
	mux.Use(otelchi.Middleware("payment-service", otelchi.WithChiRoutes(mux)))
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	for _, option := range options {
		option(r)
	}

	r.mapHealthRoutes(mux)
	r.mapPaymentsRouter(mux)

	return mux
}

func (r router) mapHealthRoutes(mux *chi.Mux) {
	if r.healthController == nil {
		panic("health controller is required")
	}
	mux.Get("/health", r.healthController.Health)
}

func (r router) mapPaymentsRouter(mux *chi.Mux) {
	if r.paymentsController == nil {
		panic("health controller is required")
	}
	mux.Route("/v1", func(v1 chi.Router) {
		if r.idempotencyMiddleware == nil {
			panic("idempotency middleware is required")
		}

		v1.Use(r.idempotencyMiddleware.Handler)
		v1.Post("/payments", r.paymentsController.CreatePayment)
	})

}
