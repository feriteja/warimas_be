package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	t.Run("Success loading from env", func(t *testing.T) {
		// t.Setenv sets the environment variable for the duration of the test
		// and automatically restores it afterwards.
		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_USER", "testuser")
		t.Setenv("DB_PASSWORD", "testpass")
		t.Setenv("DB_NAME", "testdb")
		t.Setenv("DB_PORT", "5432")
		t.Setenv("APP_PORT", "8080")
		t.Setenv("XENDIT_APIKEY", "xendit_secret")
		t.Setenv("APP_ENV", "test")

		cfg := LoadConfig()

		assert.NotNil(t, cfg)
		assert.Equal(t, "localhost", cfg.DBHost)
		assert.Equal(t, "testuser", cfg.DBUser)
		assert.Equal(t, "testpass", cfg.DBPassword)
		assert.Equal(t, "testdb", cfg.DBName)
		assert.Equal(t, "5432", cfg.DBPort)
		assert.Equal(t, "8080", cfg.AppPort)
		assert.Equal(t, "xendit_secret", cfg.XenditSecretKey)
		assert.Equal(t, "test", cfg.AppEnv)
	})
}
