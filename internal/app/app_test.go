package app_test

import (
	"pho/internal/app"
	"pho/internal/logging"
	"pho/internal/render"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	app := app.New()
	require.NotNil(t, app)
	cmd := app.GetCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "pho", cmd.Name)
	assert.Equal(t, "MongoDB document editor - query, edit, and apply changes interactively", cmd.Usage)
	assert.Len(t, cmd.Commands, 4)
}

func TestParseExtJSONMode(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		expected    render.ExtJSONMode
		expectError bool
	}{
		{
			name:        "canonical mode",
			mode:        "canonical",
			expected:    render.ExtJSONModes.Canonical,
			expectError: false,
		},
		{
			name:        "relaxed mode",
			mode:        "relaxed",
			expected:    render.ExtJSONModes.Relaxed,
			expectError: false,
		},
		{
			name:        "shell mode",
			mode:        "shell",
			expected:    render.ExtJSONModes.Shell,
			expectError: false,
		},
		{
			name:        "invalid mode",
			mode:        "invalid",
			expected:    render.ExtJSONModes.Canonical,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := app.ParseExtJSONMode(tt.mode)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid extjson-mode")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds",
			duration: 30 * time.Second,
			expected: "30 seconds",
		},
		{
			name:     "minutes",
			duration: 5 * time.Minute,
			expected: "5 minutes",
		},
		{
			name:     "hours",
			duration: 2 * time.Hour,
			expected: "2.0 hours",
		},
		{
			name:     "days",
			duration: 48 * time.Hour,
			expected: "2.0 days",
		},
		{
			name:     "fractional minutes",
			duration: 90 * time.Second,
			expected: "2 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrepareMongoURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		host     string
		port     string
		expected string
	}{
		{
			name:     "full URI provided",
			uri:      "mongodb://user:pass@host:27017/db",
			host:     "",
			port:     "",
			expected: "mongodb://user:pass@host:27017/db",
		},
		{
			name:     "host and port provided",
			uri:      "",
			host:     "myhost",
			port:     "27018",
			expected: "mongodb://myhost:27018",
		},
		{
			name:     "host only provided",
			uri:      "",
			host:     "myhost",
			port:     "",
			expected: "mongodb://myhost:27017",
		},
		{
			name:     "nothing provided",
			uri:      "",
			host:     "",
			port:     "",
			expected: "mongodb://localhost:27017",
		},
		{
			name:     "URI without mongodb prefix",
			uri:      "localhost:27017",
			host:     "",
			port:     "",
			expected: "mongodb://localhost:27017",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.PrepareMongoURI(tt.uri, tt.host, tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetVerbosityLevel(t *testing.T) {
	tests := []struct {
		name     string
		verbose  bool
		quiet    bool
		expected logging.VerbosityLevel
	}{
		{
			name:     "normal level",
			verbose:  false,
			quiet:    false,
			expected: logging.LevelNormal,
		},
		{
			name:     "verbose level",
			verbose:  true,
			quiet:    false,
			expected: logging.LevelVerbose,
		},
		{
			name:     "quiet level",
			verbose:  false,
			quiet:    true,
			expected: logging.LevelQuiet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := &mockCLICommand{
				verboseValue: tt.verbose,
				quietValue:   tt.quiet,
			}

			result := app.GetVerbosityLevel(mockCmd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateLogger(t *testing.T) {
	mockCmd := &mockCLICommand{
		verboseValue: true,
		quietValue:   false,
	}

	logger := app.CreateLogger(mockCmd)
	require.NotNil(t, logger)
	assert.Equal(t, logging.LevelVerbose, logger.GetLevel())
}

func TestGetConnectionFlags(t *testing.T) {
	flags := app.GetConnectionFlags()
	assert.Len(t, flags, 5)

	flagNames := make([]string, len(flags))
	for i, flag := range flags {
		flagNames[i] = flag.Names()[0]
	}

	expectedFlags := []string{"uri", "host", "port", "db", "collection"}
	for _, expected := range expectedFlags {
		assert.Contains(t, flagNames, expected)
	}
}

func TestGetCommonFlags(t *testing.T) {
	flags := app.GetCommonFlags()
	assert.Len(t, flags, 16) // 5 connection flags + 11 query flags

	flagNames := make([]string, len(flags))
	for i, flag := range flags {
		flagNames[i] = flag.Names()[0]
	}

	expectedFlags := []string{
		"uri", "host", "port", "db", "collection", // connection flags
		"query", "limit", "sort", "projection", "editor", "edit", "extjson-mode", "compact", "line-numbers", "verbose", "quiet", // query flags
	}
	for _, expected := range expectedFlags {
		assert.Contains(t, flagNames, expected, "Flag %s should be present", expected)
	}
}

// mockCLICommand is a mock implementation that satisfies the CLI command interface for testing.
type mockCLICommand struct {
	verboseValue bool
	quietValue   bool
	stringValues map[string]string
	int64Values  map[string]int64
	boolValues   map[string]bool
}

func (m *mockCLICommand) Bool(name string) bool {
	switch name {
	case "verbose":
		return m.verboseValue
	case "quiet":
		return m.quietValue
	default:
		if m.boolValues != nil {
			return m.boolValues[name]
		}
		return false
	}
}

func (m *mockCLICommand) String(name string) string {
	if m.stringValues != nil {
		return m.stringValues[name]
	}
	return ""
}

func (m *mockCLICommand) Int64(name string) int64 {
	if m.int64Values != nil {
		return m.int64Values[name]
	}
	return 0
}
