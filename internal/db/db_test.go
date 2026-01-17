package db

import (
	"testing"
	"warimas-be/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestBuildDSN(t *testing.T) {
	cfg := &config.Config{
		DBHost:     "localhost",
		DBUser:     "testuser",
		DBPassword: "testpassword",
		DBName:     "testdb",
		DBPort:     "5432",
	}

	expected := "host=localhost user=testuser password=testpassword dbname=testdb port=5432 sslmode=disable"
	actual := buildDSN(cfg)

	assert.Equal(t, expected, actual)
}
