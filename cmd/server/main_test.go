package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"warimas-be/internal/graph"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/assert"
)

func TestSetupRouter(t *testing.T) {
	// 1. Setup dependencies
	// We use an empty resolver since we are only testing the HTTP wiring, not the GraphQL logic itself.
	resolver := &graph.Resolver{}

	// Assuming graph.NewSchema exists as per your main.go.
	// If it's NewExecutableSchema in generated code, adjust accordingly.
	srv := handler.NewDefaultServer(graph.NewSchema(resolver))

	// Mock webhook handler
	mockWebhookHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("webhook received"))
	}

	// 2. Create Router
	router := setupRouter(srv, mockWebhookHandler)

	// 3. Test /health
	t.Run("Health Check", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "OK")
	})

	// 4. Test / (Playground)
	t.Run("GraphQL Playground", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "GraphQL Playground")
	})

	// 5. Test Webhook Wiring
	t.Run("Payment Webhook", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/webhook/payment", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "webhook received", rr.Body.String())
	})
}
