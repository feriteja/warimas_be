package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractAccessToken(t *testing.T) {
	t.Run("Cookie Preferred", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie_token"})
		// Add header as well to ensure cookie takes precedence
		req.Header.Set("Authorization", "Bearer header_token")

		token := ExtractAccessToken(req)
		assert.Equal(t, "cookie_token", token)
	})

	t.Run("Header Fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer header_token")

		token := ExtractAccessToken(req)
		assert.Equal(t, "header_token", token)
	})

	t.Run("Empty Cookie Falls Back to Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: ""})
		req.Header.Set("Authorization", "Bearer header_token")

		token := ExtractAccessToken(req)
		assert.Equal(t, "header_token", token)
	})

	t.Run("No Token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		token := ExtractAccessToken(req)
		assert.Empty(t, token)
	})

	t.Run("Malformed Header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Basic user:pass")

		token := ExtractAccessToken(req)
		assert.Empty(t, token)
	})
}
