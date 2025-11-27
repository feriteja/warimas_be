package logger

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFrom(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// FromCtx returns logger with request_id automatically added
func FromCtx(ctx context.Context) *zap.Logger {
	reqID := RequestIDFrom(ctx)
	if reqID == "" {
		return L()
	}
	return L().With(zap.String("request_id", reqID))
}
