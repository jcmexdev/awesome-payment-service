package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jcmexdev/payment-service/internal/infra/http/handler"
	appmiddleware "github.com/jcmexdev/payment-service/internal/infra/http/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/riandyrn/otelchi"
)

type router struct {
	healthController      handler.HealthCheckController
	paymentsController    handler.PaymentsController
	idempotencyMiddleware *appmiddleware.IdempotencyMiddleware
	accountController     handler.AccountController
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

func WithAccountController(accountController handler.AccountController) Options {
	return func(r *router) {
		r.accountController = accountController
	}
}

func NewRouter(options ...Options) *chi.Mux {
	r := &router{}
	mux := chi.NewRouter()
	mux.Use(otelchi.Middleware("payment_service", otelchi.WithChiRoutes(mux)))
	mux.Use(appmiddleware.TelemetryMiddleware("payment-service"))
	mux.Use(appmiddleware.PrometheusMetricsMiddleware)
	mux.Use(middleware.Recoverer)
	for _, option := range options {
		option(r)
	}

	mux.Handle("/metrics", promhttp.Handler())

	r.bindRoutes(mux)

	return mux
}

func (r router) bindRoutes(mux *chi.Mux) {
	if r.idempotencyMiddleware == nil {
		panic("idempotency middleware is required")
	}

	mux.Route("/v1", func(v1 chi.Router) {
		r.bindHealthRoutes(v1)
		r.bindAccountRoutes(v1)
		r.bindPaymentsRoutes(v1)
	})
}

func (r router) bindHealthRoutes(v1 chi.Router) {
	if r.healthController == nil {
		panic("health controller is required")
	}
	v1.Get("/health", r.healthController.Health)
}

func (r router) bindPaymentsRoutes(v1 chi.Router) {
	if r.paymentsController == nil {
		panic("health controller is required")
	}

	v1.With(r.idempotencyMiddleware.WithPrefix("payment").Handler).Post("/payments", r.paymentsController.CreatePayment)
}

func (r router) bindAccountRoutes(v1 chi.Router) {
	if r.accountController == nil {
		panic("account controller is required")
	}

	v1.With(r.idempotencyMiddleware.WithPrefix("account").Handler).Post("/accounts", r.accountController.CreateAccount)
}
