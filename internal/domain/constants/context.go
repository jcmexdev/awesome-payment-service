package constants

type ContextKey string

const (
	ContextKeySimulateDelay ContextKey = "simulate_delay"
	ContextKeySimulateError ContextKey = "simulate_error"
	ContextKeyRequestID     ContextKey = "request_id"
	ContextKeyTraceID       ContextKey = "trace_id"
)
