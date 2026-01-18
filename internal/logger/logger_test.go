package logger

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestInit(t *testing.T) {
	// Save original logger to restore later
	originalLog := log
	defer func() { log = originalLog }()

	t.Run("Production", func(t *testing.T) {
		Init("production")
		assert.NotNil(t, log)
	})

	t.Run("Development", func(t *testing.T) {
		Init("development")
		assert.NotNil(t, log)
	})
}

func TestL(t *testing.T) {
	// Save original logger
	originalLog := log
	defer func() { log = originalLog }()

	// Force nil to test lazy initialization
	log = nil
	os.Setenv("APP_ENV", "test")

	l := L()
	assert.NotNil(t, l)
	assert.NotNil(t, log)
}

func TestContextFunctions(t *testing.T) {
	ctx := context.Background()
	reqID := "test-request-id-123"

	t.Run("WithRequestID", func(t *testing.T) {
		newCtx := WithRequestID(ctx, reqID)
		assert.NotEqual(t, ctx, newCtx)

		// Verify the value is stored with the correct key
		val := newCtx.Value(requestIDKey)
		assert.Equal(t, reqID, val)
	})

	t.Run("RequestIDFrom", func(t *testing.T) {
		// Case 1: Context has Request ID
		ctxWithID := WithRequestID(ctx, reqID)
		extractedID := RequestIDFrom(ctxWithID)
		assert.Equal(t, reqID, extractedID)

		// Case 2: Context does not have Request ID
		emptyID := RequestIDFrom(ctx)
		assert.Equal(t, "", emptyID)
	})
}

func TestFromCtx(t *testing.T) {
	// Create an observer to verify logs
	core, observed := observer.New(zapcore.InfoLevel)
	obsLogger := zap.New(core)

	// Swap the global logger with our observer logger
	originalLog := log
	log = obsLogger
	defer func() { log = originalLog }()

	t.Run("WithRequestID", func(t *testing.T) {
		reqID := "req-abc-123"
		ctx := WithRequestID(context.Background(), reqID)

		// Get logger from context
		l := FromCtx(ctx)
		l.Info("test message with id")

		// Verify log output
		logs := observed.TakeAll()
		assert.Len(t, logs, 1)
		assert.Equal(t, "test message with id", logs[0].Message)

		// Verify request_id field is present
		fields := logs[0].ContextMap()
		assert.Equal(t, reqID, fields["request_id"])
	})

	t.Run("WithoutRequestID", func(t *testing.T) {
		ctx := context.Background()

		// Get logger from context
		l := FromCtx(ctx)
		l.Info("test message without id")

		// Verify log output
		logs := observed.TakeAll()
		assert.Len(t, logs, 1)
		assert.Equal(t, "test message without id", logs[0].Message)

		// Verify request_id field is NOT present
		fields := logs[0].ContextMap()
		_, ok := fields["request_id"]
		assert.False(t, ok)
	})
}

func TestSync(t *testing.T) {
	// Just ensure it doesn't panic.
	assert.NotPanics(t, func() {
		Sync()
	})
}

func TestRequestIDMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Request ID is injected into context
		rid := RequestIDFrom(r.Context())
		assert.NotEmpty(t, rid)
	})

	handler := RequestIDMiddleware(nextHandler)

	t.Run("Generates ID when missing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	})

	t.Run("Preserves existing ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		existingID := "test-id-123"
		req.Header.Set("X-Request-ID", existingID)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, existingID, w.Header().Get("X-Request-ID"))
	})
}

func TestLoggingMiddleware(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	obsLogger := zap.New(core)

	originalLog := log
	log = obsLogger
	defer func() { log = originalLog }()

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := LoggingMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	logs := observed.TakeAll()
	assert.Len(t, logs, 1)
	assert.Equal(t, "incoming request", logs[0].Message)
	assert.Equal(t, "/test", logs[0].ContextMap()["path"])
}
