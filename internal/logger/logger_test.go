package logger

import (
	"context"
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
	// Ensure L() returns a non-nil logger
	l := L()
	assert.NotNil(t, l)
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
