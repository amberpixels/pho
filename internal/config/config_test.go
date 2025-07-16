package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"pho/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefault(t *testing.T) {
	cfg := config.NewDefault()

	// MongoDB config
	assert.Equal(t, "mongodb://localhost:27017", cfg.Mongo.URI)
	assert.Equal(t, "canonical", cfg.Mongo.ExtJSONMode)

	// PostgreSQL config
	assert.Equal(t, "localhost", cfg.Postgres.Host)
	assert.Equal(t, "5432", cfg.Postgres.Port)
	assert.Equal(t, "public", cfg.Postgres.Schema)
	assert.Equal(t, "prefer", cfg.Postgres.SSLMode)

	// Database selection
	assert.Equal(t, "mongodb", cfg.Database.Type)

	// Query config
	assert.Equal(t, "{}", cfg.Query.Query)
	assert.Equal(t, int64(10000), cfg.Query.Limit)

	// App config
	assert.Equal(t, "vim", cfg.App.Editor)
	assert.Equal(t, "60s", cfg.App.Timeout)

	// Output config
	assert.Equal(t, "json", cfg.Output.Format)
	assert.True(t, cfg.Output.LineNumbers)
	assert.False(t, cfg.Output.Compact)
	assert.False(t, cfg.Output.Verbose)
	assert.False(t, cfg.Output.Quiet)
}

func TestConfig_SetAndGet(t *testing.T) {
	cfg := config.NewDefault()

	tests := []struct {
		key      string
		setValue string
		getValue interface{}
	}{
		{"mongo.uri", "mongodb://example.com:27017", "mongodb://example.com:27017"},
		{"mongo.database", "testdb", "testdb"},
		{"mongo.extjson_mode", "relaxed", "relaxed"},
		{"postgres.host", "localhost", "localhost"},
		{"postgres.database", "testdb", "testdb"},
		{"database.type", "postgres", "postgres"},
		{"query.query", "{\"test\": 1}", "{\"test\": 1}"},
		{"query.limit", "5000", int64(5000)},
		{"app.editor", "nano", "nano"},
		{"app.timeout", "30s", "30s"},
		{"output.format", "yaml", "yaml"},
		{"output.line_numbers", "false", false},
		{"output.compact", "true", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := cfg.Set(tt.key, tt.setValue)
			require.NoError(t, err)

			value, err := cfg.Get(tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.getValue, value)
		})
	}
}

func TestConfig_SetInvalidValues(t *testing.T) {
	cfg := config.NewDefault()

	tests := []struct {
		key   string
		value string
	}{
		{"app.timeout", "invalid"},
		{"query.limit", "not-a-number"},
		{"mongo.extjson_mode", "invalid"},
		{"output.format", "invalid"},
		{"database.type", "invalid"},
		{"output.line_numbers", "not-bool"},
		{"unknown.key", "value"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := cfg.Set(tt.key, tt.value)
			assert.Error(t, err)
		})
	}
}

func TestConfig_GetInvalidKey(t *testing.T) {
	cfg := config.NewDefault()

	_, err := cfg.Get("unknown.key")
	assert.Error(t, err)
}

func TestConfig_SaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Set PHO_CONFIG_DIR to temp directory
	t.Setenv("PHO_CONFIG_DIR", tempDir)

	// Create and save config
	cfg := config.NewDefault()
	cfg.Mongo.URI = "mongodb://test:27017"
	cfg.App.Editor = "emacs"
	cfg.Output.Format = "yaml"

	err := cfg.Save()
	require.NoError(t, err)

	// Verify file was created
	configPath := filepath.Join(tempDir, "config.json")
	assert.FileExists(t, configPath)

	// Load config
	loadedCfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "mongodb://test:27017", loadedCfg.Mongo.URI)
	assert.Equal(t, "emacs", loadedCfg.App.Editor)
	assert.Equal(t, "yaml", loadedCfg.Output.Format)
}

func TestConfig_EnvironmentOverrides(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Set test environment variables
	envVars := map[string]string{
		"PHO_CONFIG_DIR": tempDir,
		"MONGODB_URI":    "mongodb://env:27017",
		"MONGODB_DB":     "envdb",
		"PHO_TIMEOUT":    "120s",
	}

	// Set environment variables
	for key, value := range envVars {
		t.Setenv(key, value)
	}

	// Create config file with different values
	configPath := filepath.Join(tempDir, "config.json")
	configData := map[string]interface{}{
		"mongo": map[string]interface{}{
			"uri":      "mongodb://file:27017",
			"database": "filedb",
		},
		"app": map[string]interface{}{
			"timeout": "60s",
		},
	}

	data, err := json.Marshal(configData)
	require.NoError(t, err)

	err = os.WriteFile(configPath, data, 0600)
	require.NoError(t, err)

	// Load config - environment should override file
	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "mongodb://env:27017", cfg.Mongo.URI) // from env
	assert.Equal(t, "envdb", cfg.Mongo.Database)          // from env
	assert.Equal(t, "120s", cfg.App.Timeout)              // from env
}

func TestConfig_LoadNonExistentFile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Set PHO_CONFIG_DIR to temp directory (no config file)
	t.Setenv("PHO_CONFIG_DIR", tempDir)

	// Load should succeed with defaults
	cfg, err := config.Load()
	require.NoError(t, err)

	// Should have default values
	assert.Equal(t, "mongodb://localhost:27017", cfg.Mongo.URI)
	assert.Equal(t, "vim", cfg.App.Editor)
}
