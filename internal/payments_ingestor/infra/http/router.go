package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	handler2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/handler"
	middleware2 "github.com/jcmexdev/payment-service/internal/payments_ingestor/infra/http/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/riandyrn/otelchi"
)

type router struct {
	healthController      handler2.HealthCheckController
	paymentsController    handler2.PaymentsController
	idempotencyMiddleware *middleware2.IdempotencyMiddleware
	accountController     handler2.AccountController
}

type Options func(r *router)

func WithIdempotencyMiddleware(idempotencyMiddleware *middleware2.IdempotencyMiddleware) Options {
	return func(r *router) {
		r.idempotencyMiddleware = idempotencyMiddleware
	}
}

func WithHealthController(checkHandler handler2.HealthCheckController) Options {
	return func(r *router) {
		r.healthController = checkHandler
	}
}

func WithPaymentsController(checkHandler handler2.PaymentsController) Options {
	return func(r *router) {
		r.paymentsController = checkHandler
	}
}

func WithAccountController(accountController handler2.AccountController) Options {
	return func(r *router) {
		r.accountController = accountController
	}
}

func NewRouter(options ...Options) *chi.Mux {
	r := &router{}
	mux := chi.NewRouter()
	mux.Use(otelchi.Middleware("payment_service", otelchi.WithChiRoutes(mux)))
	mux.Use(middleware2.TelemetryMiddleware("payment-service"))
	mux.Use(middleware2.PrometheusMetricsMiddleware)
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

	v1.With(r.idempotencyMiddleware.WithPrefix("payment").Handler).Post("/payments", r.paymentsController.AuthorizePayment)
}

func (r router) bindAccountRoutes(v1 chi.Router) {
	if r.accountController == nil {
		panic("account controller is required")
	}

	v1.With(r.idempotencyMiddleware.WithPrefix("account").Handler).Post("/accounts", r.accountController.CreateAccount)
}
