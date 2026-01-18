package db

import (
	"database/sql"
	"database/sql/driver"
	"os"
	"os/exec"
	"testing"
	"warimas-be/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestBuildDSN(t *testing.T) {
	cfg := &config.Config{
		DBHost:     "localhost",
		DBUser:     "test_user",
		DBPassword: "test_password",
		DBName:     "test_db",
		DBPort:     "5432",
	}

	expected := "host=localhost user=test_user password=test_password dbname=test_db port=5432 sslmode=disable"
	result := buildDSN(cfg)

	assert.Equal(t, expected, result)
}

func TestNewDatabase_ConnectionFailure(t *testing.T) {
	// This test ensures that NewDatabase returns an error (and doesn't crash)
	// when it cannot connect to the database (Ping fails).
	cfg := &config.Config{
		DBHost: "invalid_host",
		DBPort: "5432",
	}

	db, err := NewDatabase(cfg)

	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to ping DB")
}

func TestNewDatabase_InvalidDriver(t *testing.T) {
	cfg := &config.Config{}
	// "invalid_driver_name" is not registered, so sql.Open will return an error
	db, err := newDatabaseWithDriver(cfg, "invalid_driver_name")

	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to connect to DB")
}

func TestInitDB_Failure(t *testing.T) {
	// This test runs the test binary as a subprocess to verify that InitDB calls log.Fatalf
	if os.Getenv("BE_CRASHER") == "1" {
		cfg := &config.Config{
			DBHost: "invalid_host",
			DBPort: "5432",
		}
		InitDB(cfg)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestInitDB_Failure")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()

	// We expect the process to exit with a non-zero status (log.Fatalf exits with 1)
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

// --- Mock Driver for Success Test ---
// This mock driver allows us to test the "happy path" of sql.Open and db.Ping
// without needing a real database running.

type mockDriver struct{}

func (m *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct{}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) { return &mockStmt{}, nil }
func (c *mockConn) Close() error                              { return nil }
func (c *mockConn) Begin() (driver.Tx, error)                 { return nil, nil }

type mockStmt struct{}

func (s *mockStmt) Close() error                                    { return nil }
func (s *mockStmt) NumInput() int                                   { return 0 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) { return nil, nil }
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error)  { return nil, nil }

func init() {
	sql.Register("mock_driver_success", &mockDriver{})
}

func TestNewDatabase_Success(t *testing.T) {
	cfg := &config.Config{DBHost: "localhost"}
	db, err := newDatabaseWithDriver(cfg, "mock_driver_success")
	assert.NoError(t, err)
	assert.NotNil(t, db)
}
