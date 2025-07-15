package logging_test

import (
	"bytes"
	"strings"
	"testing"

	"pho/internal/logging"

	"github.com/stretchr/testify/assert"
)

func TestVerbosityLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    logging.VerbosityLevel
		expected string
	}{
		{"quiet level", logging.LevelQuiet, "quiet"},
		{"normal level", logging.LevelNormal, "normal"},
		{"verbose level", logging.LevelVerbose, "verbose"},
		{"unknown level", logging.VerbosityLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestNewLogger(t *testing.T) {
	logger := logging.NewLogger(logging.LevelVerbose)
	assert.NotNil(t, logger)
	assert.Equal(t, logging.LevelVerbose, logger.GetLevel())
	assert.True(t, logger.IsVerbose())
	assert.False(t, logger.IsQuiet())
}

func TestLogger_SetOutput(t *testing.T) {
	logger := logging.NewLogger(logging.LevelNormal)
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.Info("test message")
	assert.Contains(t, buf.String(), "test message")
}

func TestLogger_SetErrorOutput(t *testing.T) {
	logger := logging.NewLogger(logging.LevelNormal)
	var buf bytes.Buffer
	logger.SetErrorOutput(&buf)

	logger.Error("test error")
	assert.Contains(t, buf.String(), "Error: test error")
}

func TestLogger_QuietMode(t *testing.T) {
	logger := logging.NewLogger(logging.LevelQuiet)
	var outputBuf, errorBuf bytes.Buffer
	logger.SetOutput(&outputBuf)
	logger.SetErrorOutput(&errorBuf)

	// In quiet mode, only fatal errors should show
	logger.Info("info message")
	logger.Verbose("verbose message")
	logger.Progress("progress message")
	logger.Success("success message")
	logger.Warning("warning message")

	// These should not appear in output
	assert.Empty(t, outputBuf.String())

	// Errors should still appear
	logger.Error("error message")
	logger.Fatal("fatal message")
	assert.Contains(t, errorBuf.String(), "error message")
	assert.Contains(t, errorBuf.String(), "fatal message")
}

func TestLogger_NormalMode(t *testing.T) {
	logger := logging.NewLogger(logging.LevelNormal)
	var outputBuf, errorBuf bytes.Buffer
	logger.SetOutput(&outputBuf)
	logger.SetErrorOutput(&errorBuf)

	logger.Info("info message")
	logger.Verbose("verbose message")
	logger.Progress("progress message")
	logger.Success("success message")
	logger.Warning("warning message")
	logger.Debug("debug message")

	output := outputBuf.String()

	// Should show info, progress, success, warning
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "progress message")
	assert.Contains(t, output, "✓ success message")
	assert.Contains(t, output, "⚠️  Warning: warning message")

	// Should NOT show verbose or debug
	assert.NotContains(t, output, "verbose message")
	assert.NotContains(t, output, "debug message")

	// Errors should appear in error output
	logger.Error("error message")
	assert.Contains(t, errorBuf.String(), "Error: error message")
}

func TestLogger_VerboseMode(t *testing.T) {
	logger := logging.NewLogger(logging.LevelVerbose)
	var outputBuf, errorBuf bytes.Buffer
	logger.SetOutput(&outputBuf)
	logger.SetErrorOutput(&errorBuf)

	logger.Info("info message")
	logger.Verbose("verbose message")
	logger.Progress("progress message")
	logger.Success("success message")
	logger.Warning("warning message")
	logger.Debug("debug message")

	output := outputBuf.String()

	// Should show all messages
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "verbose message")
	assert.Contains(t, output, "progress message")
	assert.Contains(t, output, "✓ success message")
	assert.Contains(t, output, "⚠️  Warning: warning message")
	assert.Contains(t, output, "DEBUG] debug message")

	// Timestamps should be present in verbose and debug messages
	assert.True(t, strings.Contains(output, "[") && strings.Contains(output, "]"))
}

func TestLogger_IsQuiet(t *testing.T) {
	tests := []struct {
		level    logging.VerbosityLevel
		expected bool
	}{
		{logging.LevelQuiet, true},
		{logging.LevelNormal, false},
		{logging.LevelVerbose, false},
	}

	for _, tt := range tests {
		logger := logging.NewLogger(tt.level)
		assert.Equal(t, tt.expected, logger.IsQuiet())
	}
}

func TestLogger_IsVerbose(t *testing.T) {
	tests := []struct {
		level    logging.VerbosityLevel
		expected bool
	}{
		{logging.LevelQuiet, false},
		{logging.LevelNormal, false},
		{logging.LevelVerbose, true},
	}

	for _, tt := range tests {
		logger := logging.NewLogger(tt.level)
		assert.Equal(t, tt.expected, logger.IsVerbose())
	}
}

func TestLogger_MessageFormatting(t *testing.T) {
	logger := logging.NewLogger(logging.LevelVerbose)
	var outputBuf bytes.Buffer
	logger.SetOutput(&outputBuf)

	// Test message formatting with arguments
	logger.Info("Processing %d documents in %s collection", 5, "users")
	assert.Contains(t, outputBuf.String(), "Processing 5 documents in users collection")

	outputBuf.Reset()
	logger.Success("Completed operation with %s result", "successful")
	assert.Contains(t, outputBuf.String(), "✓ Completed operation with successful result")
}
