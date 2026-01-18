package main

import (
	"database/sql"
	"database/sql/driver"
	"net/http"
	"net/http/httptest"
	"testing"

	"warimas-be/internal/config"
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

func TestNewServer(t *testing.T) {
	// 1. Setup Mock DB
	// We use a mock driver so we don't need a real Postgres connection
	db, err := sql.Open("mock_driver_main", "")
	assert.NoError(t, err)

	// 2. Setup Config
	cfg := &config.Config{
		AppPort:         "8080",
		AppEnv:          "test",
		XenditSecretKey: "dummy_secret",
	}

	// 3. Call newServer (The function we want to cover)
	router := newServer(cfg, db)

	// 4. Assertions
	assert.NotNil(t, router)
	// Verify that the router handles the expected paths
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- Mock Driver for Testing ---
type mockDriver struct{}

func (m *mockDriver) Open(name string) (driver.Conn, error)         { return &mockConn{}, nil }
func (c *mockConn) Prepare(query string) (driver.Stmt, error)       { return &mockStmt{}, nil }
func (c *mockConn) Close() error                                    { return nil }
func (c *mockConn) Begin() (driver.Tx, error)                       { return nil, nil }
func (s *mockStmt) Close() error                                    { return nil }
func (s *mockStmt) NumInput() int                                   { return 0 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) { return nil, nil }
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error)  { return nil, nil }

type mockConn struct{}
type mockStmt struct{}

func init() {
	sql.Register("mock_driver_main", &mockDriver{})
}

func TestRun(t *testing.T) {
	// 1. Mock initDBFunc
	origInitDB := initDBFunc
	defer func() { initDBFunc = origInitDB }()
	initDBFunc = func(cfg *config.Config) *sql.DB {
		db, _ := sql.Open("mock_driver_main", "")
		return db
	}

	// 2. Mock startServerFunc
	origStartServer := startServerFunc
	defer func() { startServerFunc = origStartServer }()
	startServerFunc = func(addr string, handler http.Handler) error {
		return nil
	}

	// 3. Set Environment
	t.Setenv("APP_PORT", "8080")
	t.Setenv("APP_ENV", "test")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "pass")
	t.Setenv("DB_NAME", "db")

	// 4. Run
	assert.NoError(t, run())
}
