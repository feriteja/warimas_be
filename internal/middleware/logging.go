package middleware

import (
	"context"
	"net/http"
	"time"

	"warimas-be/internal/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const loggerKey contextKey = "requestLogger"

// L extracts logger from context
func L(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return l
	}
	return logger.L()
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		// Create logger bound to this request
		reqLogger := logger.L().With(
			zap.String("request_id", reqID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)

		// Put logger into request context
		ctx := context.WithValue(r.Context(), loggerKey, reqLogger)
		ctx = logger.WithRequestID(ctx, reqID)
		r = r.WithContext(ctx)

		// Continue
		next.ServeHTTP(w, r)

		// Log request summary
		reqLogger.Info("request completed",
			zap.Duration("duration", time.Since(start)),
			zap.String("ip", r.RemoteAddr),
		)
	})
}
