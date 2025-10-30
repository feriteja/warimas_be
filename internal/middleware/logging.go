package middleware

import (
	"net/http"
	"time"
	"warimas-be/internal/logger"
	"warimas-be/internal/utils"
)

// responseRecorder lets us capture HTTP status codes
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs every HTTP request in structured JSON
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// capture response status
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)

		duration := time.Since(start)
		userID, _ := utils.GetUserIDFromContext(r.Context())

		logger.Info("HTTP Request", map[string]interface{}{
			"method":   r.Method,
			"path":     r.URL.Path,
			"status":   rec.statusCode,
			"duration": duration.String(),
			"remoteIP": r.RemoteAddr,
			"userID":   userID,
		})
	})
}
