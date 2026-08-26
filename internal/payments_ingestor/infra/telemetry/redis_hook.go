package telemetry

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OpenTelemetryRedisHook implements redis.Hook to trace Redis commands using OpenTelemetry.
type OpenTelemetryRedisHook struct {
	tracer trace.Tracer
}

// NewOpenTelemetryRedisHook creates a new instance of OpenTelemetryRedisHook.
func NewOpenTelemetryRedisHook() *OpenTelemetryRedisHook {
	return &OpenTelemetryRedisHook{
		tracer: otel.Tracer("redis-instrumentation"),
	}
}

// DialHook intercepts connection dials.
func (h *OpenTelemetryRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook intercepts single commands.
func (h *OpenTelemetryRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		// Format span name as redis.<command>
		spanName := "redis." + cmd.Name()

		ctx, span := h.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()

		span.SetAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", cmd.Name()),
			attribute.String("db.statement", cmd.String()),
		)

		err := next(ctx, cmd)
		if err != nil && err != redis.Nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}

// ProcessPipelineHook intercepts pipeline commands.
func (h *OpenTelemetryRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		ctx, span := h.tracer.Start(ctx, "redis.pipeline", trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()

		span.SetAttributes(
			attribute.String("db.system", "redis"),
			attribute.Int("db.redis.num_cmd", len(cmds)),
		)

		err := next(ctx, cmds)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}
