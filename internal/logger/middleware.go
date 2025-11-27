package logger

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		ctx := WithRequestID(r.Context(), reqID)
		w.Header().Set("X-Request-ID", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log := FromCtx(r.Context())

		next.ServeHTTP(w, r)

		log.Info("incoming request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("ip", r.RemoteAddr),
			zap.Duration("duration_ms", time.Since(start)),
		)
	})
}
