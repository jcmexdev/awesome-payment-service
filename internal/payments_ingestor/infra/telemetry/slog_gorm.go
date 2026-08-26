package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

// GormSlogLogger is a custom GORM logger implementation using the standard slog package.
type GormSlogLogger struct {
	logger        *slog.Logger
	logLevel      gormlogger.LogLevel
	slowThreshold time.Duration
}

// NewGormSlogLogger creates a new GormSlogLogger configured with the provided slog.Logger and slow query threshold.
func NewGormSlogLogger(l *slog.Logger, slowThreshold time.Duration) *GormSlogLogger {
	if l == nil {
		l = slog.Default()
	}
	return &GormSlogLogger{
		logger:        l.With(slog.String("component", "gorm")),
		logLevel:      gormlogger.Info,
		slowThreshold: slowThreshold,
	}
}

// LogMode sets the logging level of the GormSlogLogger.
func (l *GormSlogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info logs messages at GORM Info level.
func (l *GormSlogLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel >= gormlogger.Info {
		l.logger.InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

// Warn logs messages at GORM Warn level.
func (l *GormSlogLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel >= gormlogger.Warn {
		l.logger.WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

// Error logs messages at GORM Error level.
func (l *GormSlogLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.logLevel >= gormlogger.Error {
		l.logger.ErrorContext(ctx, fmt.Sprintf(msg, args...))
	}
}

// Trace logs SQL statements, execution duration, and affected rows.
func (l *GormSlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.logLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []any{
		slog.String("sql", sql),
		slog.Duration("duration", elapsed),
	}

	if rows >= 0 {
		fields = append(fields, slog.Int64("rows", rows))
	}

	if err != nil && !errors.Is(err, gormlogger.ErrRecordNotFound) {
		fields = append(fields, slog.String("error", err.Error()))
		l.logger.ErrorContext(ctx, "database_query_error", fields...)
		return
	}

	if l.slowThreshold != 0 && elapsed > l.slowThreshold {
		l.logger.WarnContext(ctx, "database_slow_query", fields...)
		return
	}

	if l.logLevel >= gormlogger.Info {
		l.logger.InfoContext(ctx, "database_query", fields...)
	}
}
