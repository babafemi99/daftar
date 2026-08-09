// Package requestctx owns typed request-scoped context values shared between
// transport middleware and application services.
package requestctx

import (
	"context"
	"log/slog"

	"github.com/babafemi99/daftar/backend/internal/model"
)

type key uint8

const (
	requestIDKey key = iota
	executorKey
	loggerKey
	requestLogKey
)

// RequestLog carries mutable, non-sensitive fields discovered after the outer
// request logger starts (for example the authenticated user ID).
type RequestLog struct {
	UserID string
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok && requestID != ""
}

func WithExecutor(ctx context.Context, executor model.Executor) context.Context {
	return context.WithValue(ctx, executorKey, executor)
}

func Executor(ctx context.Context) (model.Executor, bool) {
	executor, ok := ctx.Value(executorKey).(model.Executor)
	return executor, ok && executor.ID != ""
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func Logger(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)
	if !ok || logger == nil {
		return slog.Default()
	}
	return logger
}

func WithRequestLog(ctx context.Context, fields *RequestLog) context.Context {
	return context.WithValue(ctx, requestLogKey, fields)
}

func RequestLogFields(ctx context.Context) (*RequestLog, bool) {
	fields, ok := ctx.Value(requestLogKey).(*RequestLog)
	return fields, ok && fields != nil
}
