package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextHelpers(t *testing.T) {
	t.Run("Success_InjectAndRetrieve", func(t *testing.T) {
		// Arrange
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		w := httptest.NewRecorder()
		ctx := context.Background()

		// Act
		ctx = WithHTTP(ctx, req, w)
		gotReq := GetRequest(ctx)
		gotW := GetResponseWriter(ctx)

		// Assert
		assert.Equal(t, req, gotReq, "Request should match the injected request")
		assert.Equal(t, w, gotW, "ResponseWriter should match the injected writer")
	})

	t.Run("Empty_Context_ReturnsNil", func(t *testing.T) {
		ctx := context.Background()

		assert.Nil(t, GetRequest(ctx), "GetRequest should return nil if key is missing")
		assert.Nil(t, GetResponseWriter(ctx), "GetResponseWriter should return nil if key is missing")
	})
}
