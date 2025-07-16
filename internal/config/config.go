package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	defaultLimit          = 10000 // Default limit for document retrieval
	defaultTimeoutSeconds = 60    // Default timeout in seconds
)

// Config represents the application configuration.
type Config struct {
	// Database-specific settings
	Mongo    MongoConfig    `toml:"mongo"`
	Postgres PostgresConfig `toml:"postgres"`

	// Database selection and generic query settings
	Database DatabaseConfig `toml:"database"`
	Query    QueryConfig    `toml:"query"`

	// Application settings
	App AppConfig `toml:"app"`

	// Output/Display settings
	Output OutputConfig `toml:"output"`

	// Directory settings
	Directories DirectoriesConfig `toml:"directories"`
}

// MongoConfig contains MongoDB-specific connection settings.
type MongoConfig struct {
	URI         string `toml:"uri"`
	Host        string `toml:"host"`
	Port        string `toml:"port"`
	Database    string `toml:"database"`
	Collection  string `toml:"collection"`
	ExtJSONMode string `toml:"extjson_mode"`
}

// PostgresConfig contains PostgreSQL-specific connection settings (future).
type PostgresConfig struct {
	Host     string `toml:"host"`
	Port     string `toml:"port"`
	Database string `toml:"database"`
	Schema   string `toml:"schema"`
	User     string `toml:"user"`
	SSLMode  string `toml:"ssl_mode"`
}

// DatabaseConfig contains database type selection.
type DatabaseConfig struct {
	Type string `toml:"type"` // "mongodb", "postgres", etc.
}

// QueryConfig contains database-agnostic query settings.
type QueryConfig struct {
	Query      string `toml:"query"`
	Limit      int64  `toml:"limit"`
	Sort       string `toml:"sort"`
	Projection string `toml:"projection"`
}

// AppConfig contains application behavior settings.
type AppConfig struct {
	Editor  string `toml:"editor"`
	Timeout string `toml:"timeout"`
}

// OutputConfig contains output formatting settings.
type OutputConfig struct {
	Format      string `toml:"format"` // "json", "yaml", "csv", etc.
	LineNumbers bool   `toml:"line_numbers"`
	Compact     bool   `toml:"compact"`
	Verbose     bool   `toml:"verbose"`
	Quiet       bool   `toml:"quiet"`
}

// DirectoriesConfig contains directory path settings.
type DirectoriesConfig struct {
	DataDir   string `toml:"data_dir"`
	ConfigDir string `toml:"config_dir"`
}

// NewDefault returns a new Config with default values.
func NewDefault() *Config {
	return &Config{
		Mongo: MongoConfig{
			URI:         "mongodb://localhost:27017",
			Host:        "",
			Port:        "",
			Database:    "",
			Collection:  "",
			ExtJSONMode: "canonical",
		},
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     "5432",
			Database: "",
			Schema:   "public",
			User:     "",
			SSLMode:  "prefer",
		},
		Database: DatabaseConfig{
			Type: "mongodb", // Default to MongoDB for backward compatibility
		},
		Query: QueryConfig{
			Query:      "{}",
			Limit:      defaultLimit,
			Sort:       "",
			Projection: "",
		},
		App: AppConfig{
			Editor:  "vim",
			Timeout: "60s",
		},
		Output: OutputConfig{
			Format:      "json",
			LineNumbers: true,
			Compact:     false,
			Verbose:     false,
			Quiet:       false,
		},
		Directories: DirectoriesConfig{
			DataDir:   "", // Will be computed dynamically if empty
			ConfigDir: "", // Will be computed dynamically if empty
		},
	}
}

// Load loads configuration from file and applies environment variable overrides.
func Load() (*Config, error) {
	config := NewDefault()

	// Load from config file if it exists
	configPath, err := getConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("could not get config file path: %w", err)
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := toml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("could not parse config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("could not read config file: %w", err)
	}

	// Apply environment variable overrides
	config.applyEnvironmentOverrides()

	return config, nil
}

// Save saves the configuration to file.
func (c *Config) Save() error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return fmt.Errorf("could not get config file path: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	return nil
}

// applyEnvironmentOverrides applies environment variable overrides to the config.
func (c *Config) applyEnvironmentOverrides() {
	// MongoDB settings
	if val := os.Getenv("MONGODB_URI"); val != "" {
		c.Mongo.URI = val
	}
	if val := os.Getenv("MONGODB_HOST"); val != "" {
		c.Mongo.Host = val
	}
	if val := os.Getenv("MONGODB_PORT"); val != "" {
		c.Mongo.Port = val
	}
	if val := os.Getenv("MONGODB_DB"); val != "" {
		c.Mongo.Database = val
	}
	if val := os.Getenv("MONGODB_COLLECTION"); val != "" {
		c.Mongo.Collection = val
	}

	// PostgreSQL settings (for future use)
	if val := os.Getenv("POSTGRES_HOST"); val != "" {
		c.Postgres.Host = val
	}
	if val := os.Getenv("POSTGRES_PORT"); val != "" {
		c.Postgres.Port = val
	}
	if val := os.Getenv("POSTGRES_DB"); val != "" {
		c.Postgres.Database = val
	}
	if val := os.Getenv("POSTGRES_USER"); val != "" {
		c.Postgres.User = val
	}
	if val := os.Getenv("POSTGRES_SCHEMA"); val != "" {
		c.Postgres.Schema = val
	}

	// Database type
	if val := os.Getenv("PHO_DATABASE_TYPE"); val != "" {
		c.Database.Type = val
	}

	// Query settings
	if val := os.Getenv("PHO_QUERY"); val != "" {
		c.Query.Query = val
	}
	if val := os.Getenv("PHO_LIMIT"); val != "" {
		if limit, err := strconv.ParseInt(val, 10, 64); err == nil {
			c.Query.Limit = limit
		}
	}
	if val := os.Getenv("PHO_SORT"); val != "" {
		c.Query.Sort = val
	}
	if val := os.Getenv("PHO_PROJECTION"); val != "" {
		c.Query.Projection = val
	}

	// App settings
	if val := os.Getenv("PHO_EDITOR"); val != "" {
		c.App.Editor = val
	}
	if val := os.Getenv("PHO_TIMEOUT"); val != "" {
		c.App.Timeout = val
	}

	// MongoDB specific
	if val := os.Getenv("PHO_EXTJSON_MODE"); val != "" {
		c.Mongo.ExtJSONMode = val
	}

	// Directory settings
	if val := os.Getenv("PHO_DATA_DIR"); val != "" {
		c.Directories.DataDir = val
	}
	if val := os.Getenv("PHO_CONFIG_DIR"); val != "" {
		c.Directories.ConfigDir = val
	}

	// Output settings
	if val := os.Getenv("PHO_OUTPUT_COMPACT"); val != "" {
		if compact, err := strconv.ParseBool(val); err == nil {
			c.Output.Compact = compact
		}
	}
	if val := os.Getenv("PHO_OUTPUT_LINE_NUMBERS"); val != "" {
		if lineNumbers, err := strconv.ParseBool(val); err == nil {
			c.Output.LineNumbers = lineNumbers
		}
	}
	if val := os.Getenv("PHO_OUTPUT_VERBOSE"); val != "" {
		if verbose, err := strconv.ParseBool(val); err == nil {
			c.Output.Verbose = verbose
		}
	}
	if val := os.Getenv("PHO_OUTPUT_QUIET"); val != "" {
		if quiet, err := strconv.ParseBool(val); err == nil {
			c.Output.Quiet = quiet
		}
	}
}

// Set sets a configuration value by key path (e.g., "mongo.uri", "app.editor").
func (c *Config) Set(key, value string) error {
	switch key {
	// MongoDB settings
	case "mongo.uri":
		c.Mongo.URI = value
	case "mongo.host":
		c.Mongo.Host = value
	case "mongo.port":
		c.Mongo.Port = value
	case "mongo.database", "mongo.db":
		c.Mongo.Database = value
	case "mongo.collection":
		c.Mongo.Collection = value
	case "mongo.extjson_mode", "mongo.extjson-mode":
		if value != "canonical" && value != "relaxed" && value != "shell" {
			return fmt.Errorf("invalid extjson mode: %s (valid: canonical, relaxed, shell)", value)
		}
		c.Mongo.ExtJSONMode = value

	// PostgreSQL settings (future)
	case "postgres.host":
		c.Postgres.Host = value
	case "postgres.port":
		c.Postgres.Port = value
	case "postgres.database", "postgres.db":
		c.Postgres.Database = value
	case "postgres.schema":
		c.Postgres.Schema = value
	case "postgres.user":
		c.Postgres.User = value
	case "postgres.ssl_mode", "postgres.ssl-mode":
		c.Postgres.SSLMode = value

	// Database selection
	case "database.type":
		if value != "mongodb" && value != "postgres" {
			return fmt.Errorf("invalid database type: %s (valid: mongodb, postgres)", value)
		}
		c.Database.Type = value

	// Query settings
	case "query.query":
		c.Query.Query = value
	case "query.limit":
		limit, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid limit value: %w", err)
		}
		c.Query.Limit = limit
	case "query.sort":
		c.Query.Sort = value
	case "query.projection":
		c.Query.Projection = value

	// App settings
	case "app.editor":
		c.App.Editor = value
	case "app.timeout":
		// Validate the duration format
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("invalid timeout duration: %w", err)
		}
		c.App.Timeout = value

	// Output settings
	case "output.format":
		// Accept common format types
		if value != "json" && value != "yaml" && value != "csv" {
			return fmt.Errorf("invalid format: %s (valid: json, yaml, csv)", value)
		}
		c.Output.Format = value
	case "output.line_numbers", "output.line-numbers":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %w", err)
		}
		c.Output.LineNumbers = val
	case "output.compact":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %w", err)
		}
		c.Output.Compact = val
	case "output.verbose":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %w", err)
		}
		c.Output.Verbose = val
	case "output.quiet":
		val, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %w", err)
		}
		c.Output.Quiet = val

	// Directory settings
	case "directories.data_dir", "directories.data-dir":
		c.Directories.DataDir = value
	case "directories.config_dir", "directories.config-dir":
		c.Directories.ConfigDir = value

	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}

// Get gets a configuration value by key path.
func (c *Config) Get(key string) (interface{}, error) {
	switch key {
	// MongoDB settings
	case "mongo.uri":
		return c.Mongo.URI, nil
	case "mongo.host":
		return c.Mongo.Host, nil
	case "mongo.port":
		return c.Mongo.Port, nil
	case "mongo.database", "mongo.db":
		return c.Mongo.Database, nil
	case "mongo.collection":
		return c.Mongo.Collection, nil
	case "mongo.extjson_mode", "mongo.extjson-mode":
		return c.Mongo.ExtJSONMode, nil

	// PostgreSQL settings (future)
	case "postgres.host":
		return c.Postgres.Host, nil
	case "postgres.port":
		return c.Postgres.Port, nil
	case "postgres.database", "postgres.db":
		return c.Postgres.Database, nil
	case "postgres.schema":
		return c.Postgres.Schema, nil
	case "postgres.user":
		return c.Postgres.User, nil
	case "postgres.ssl_mode", "postgres.ssl-mode":
		return c.Postgres.SSLMode, nil

	// Database selection
	case "database.type":
		return c.Database.Type, nil

	// Query settings
	case "query.query":
		return c.Query.Query, nil
	case "query.limit":
		return c.Query.Limit, nil
	case "query.sort":
		return c.Query.Sort, nil
	case "query.projection":
		return c.Query.Projection, nil

	// App settings
	case "app.editor":
		return c.App.Editor, nil
	case "app.timeout":
		return c.App.Timeout, nil

	// Output settings
	case "output.format":
		return c.Output.Format, nil
	case "output.line_numbers", "output.line-numbers":
		return c.Output.LineNumbers, nil
	case "output.compact":
		return c.Output.Compact, nil
	case "output.verbose":
		return c.Output.Verbose, nil
	case "output.quiet":
		return c.Output.Quiet, nil

	// Directory settings
	case "directories.data_dir", "directories.data-dir":
		return c.Directories.DataDir, nil
	case "directories.config_dir", "directories.config-dir":
		return c.Directories.ConfigDir, nil

	default:
		return nil, fmt.Errorf("unknown config key: %s", key)
	}
}

// getConfigFilePath returns the path to the configuration file.
func getConfigFilePath() (string, error) {
	configDir := os.Getenv("PHO_CONFIG_DIR")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not get user home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config", "pho")
	}

	return filepath.Join(configDir, "config.toml"), nil
}

// GetTimeoutDuration returns the timeout as a time.Duration.
func (c *Config) GetTimeoutDuration() time.Duration {
	if timeout, err := time.ParseDuration(c.App.Timeout); err == nil {
		return timeout
	}
	// Fallback to default if parsing fails
	return defaultTimeoutSeconds * time.Second
}
