package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("CFG_TEST_KEY", "nilai")
	assert.Equal(t, "nilai", getEnv("CFG_TEST_KEY", "default"))

	t.Setenv("CFG_TEST_KEY", "")
	assert.Equal(t, "default", getEnv("CFG_TEST_KEY", "default"))

	assert.Equal(t, "default", getEnv("CFG_TEST_TIDAK_ADA", "default"))
}

func TestLoadConfigDefaults(t *testing.T) {
	for _, k := range []string{"APP_ENV", "APP_PORT", "APP_KEY", "ADMIN_USERNAME", "ADMIN_PASSWORD", "DB_HOST", "DB_PORT",
		"DB_DATABASE", "DB_USERNAME", "DB_PASSWORD", "DB_SSLMODE", "SEARCH_ENGINE_ENDPOINT", "OPENAI_ENDPOINT", "OPENAI_KEY", "OPENAI_MODEL"} {
		t.Setenv(k, "")
	}
	cfg := LoadConfig()
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "pabwe_p9_blog", cfg.DBDatabase)
	assert.Equal(t, "https://delcom.org/api/v1/search", cfg.SearchEngineEndpoint)
}

func TestLoadConfigFromEnvAndDSN(t *testing.T) {
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_HOST", "db.local")
	cfg := LoadConfig()
	assert.Equal(t, "9090", cfg.AppPort)
	assert.Contains(t, cfg.DSN(), "host=db.local")
	assert.Contains(t, cfg.DSN(), "dbname=pabwe_p9_blog")
	assert.Contains(t, cfg.DSN(), "sslmode=disable")
}
