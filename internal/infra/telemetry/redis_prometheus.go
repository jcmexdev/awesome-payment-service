package telemetry

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

var (
	redisCommandsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_commands_total",
			Help: "Total number of Redis commands executed.",
		},
		[]string{"command", "status"}, // status: success, error, hit, miss
	)
	redisCommandDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_command_duration_seconds",
			Help:    "Latency of Redis commands in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command"},
	)
)

func init() {
	prometheus.MustRegister(redisCommandsTotal)
	prometheus.MustRegister(redisCommandDuration)
}

type PrometheusRedisHook struct{}

func NewPrometheusRedisHook() *PrometheusRedisHook {
	return &PrometheusRedisHook{}
}

func (h *PrometheusRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *PrometheusRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		duration := time.Since(start).Seconds()

		commandName := cmd.Name()
		redisCommandDuration.WithLabelValues(commandName).Observe(duration)

		status := "success"
		if err != nil {
			if err == redis.Nil {
				status = "miss" // Key didn't exist
			} else {
				status = "error"
			}
		} else if commandName == "get" || commandName == "hget" {
			status = "hit"
		} else if commandName == "set" {
			// Distinguish SetNX (BoolCmd) from Set (StatusCmd) for idempotency hit/miss tracking
			if boolCmd, ok := cmd.(*redis.BoolCmd); ok {
				if boolCmd.Val() {
					status = "miss" // Key did not exist, lock acquired (idempotency miss)
				} else {
					status = "hit"  // Key already existed, lock failed (idempotency hit)
				}
			}
		}

		redisCommandsTotal.WithLabelValues(commandName, status).Inc()
		return err
	}
}

func (h *PrometheusRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		duration := time.Since(start).Seconds()

		redisCommandDuration.WithLabelValues("pipeline").Observe(duration)

		for _, cmd := range cmds {
			status := "success"
			if err != nil && err != redis.Nil {
				status = "error"
			}
			redisCommandsTotal.WithLabelValues(cmd.Name(), status).Inc()
		}
		return err
	}
}
