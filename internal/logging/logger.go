package logging

import (
	"fmt"
	"io"
	"os"
	"time"
)

// VerbosityLevel represents the verbosity level for logging
type VerbosityLevel int

const (
	// LevelQuiet suppresses all non-essential output
	LevelQuiet VerbosityLevel = iota
	// LevelNormal shows standard output (default)
	LevelNormal
	// LevelVerbose shows detailed progress and debug information
	LevelVerbose
)

// String returns the string representation of the verbosity level
func (v VerbosityLevel) String() string {
	switch v {
	case LevelQuiet:
		return "quiet"
	case LevelNormal:
		return "normal"
	case LevelVerbose:
		return "verbose"
	default:
		return "unknown"
	}
}

// Logger provides verbosity-aware logging functionality
type Logger struct {
	level  VerbosityLevel
	output io.Writer
	errors io.Writer
}

// NewLogger creates a new logger with the specified verbosity level
func NewLogger(level VerbosityLevel) *Logger {
	return &Logger{
		level:  level,
		output: os.Stdout,
		errors: os.Stderr,
	}
}

// SetOutput sets the output writer for info messages
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
}

// SetErrorOutput sets the output writer for error messages
func (l *Logger) SetErrorOutput(w io.Writer) {
	l.errors = w
}

// GetLevel returns the current verbosity level
func (l *Logger) GetLevel() VerbosityLevel {
	return l.level
}

// IsQuiet returns true if the logger is in quiet mode
func (l *Logger) IsQuiet() bool {
	return l.level == LevelQuiet
}

// IsVerbose returns true if the logger is in verbose mode
func (l *Logger) IsVerbose() bool {
	return l.level == LevelVerbose
}

// Info logs an informational message (shown in normal and verbose modes)
func (l *Logger) Info(format string, args ...any) {
	if l.level < LevelNormal {
		return
	}
	fmt.Fprintf(l.output, format+"\n", args...)
}

// Verbose logs a verbose message (only shown in verbose mode)
func (l *Logger) Verbose(format string, args ...any) {
	if l.level < LevelVerbose {
		return
	}

	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(l.output, "[%s] %s\n", timestamp, fmt.Sprintf(format, args...))
}

// Error logs an error message (always shown unless in quiet mode for non-critical errors)
func (l *Logger) Error(format string, args ...any) {
	fmt.Fprintf(l.errors, "Error: %s\n", fmt.Sprintf(format, args...))
}

// Fatal logs a fatal error message and is always shown
func (l *Logger) Fatal(format string, args ...any) {
	fmt.Fprintf(l.errors, "Fatal: %s\n", fmt.Sprintf(format, args...))
}

// Progress logs a progress message (shown in normal and verbose modes)
func (l *Logger) Progress(format string, args ...any) {
	if l.level < LevelNormal {
		return
	}

	fmt.Fprintf(l.output, format, args...)
}

// Debug logs a debug message (only shown in verbose mode)
func (l *Logger) Debug(format string, args ...any) {
	if l.level < LevelVerbose {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	fmt.Fprintf(l.output, "[%s DEBUG] %s\n", timestamp, fmt.Sprintf(format, args...))
}

// Success logs a success message (shown in normal and verbose modes)
func (l *Logger) Success(format string, args ...any) {
	if l.level < LevelNormal {
		return
	}

	fmt.Fprintf(l.output, "✓ %s\n", fmt.Sprintf(format, args...))
}

// Warning logs a warning message (shown in normal and verbose modes)
func (l *Logger) Warning(format string, args ...any) {
	if l.level < LevelNormal {
		return
	}
	fmt.Fprintf(l.output, "⚠ Warning: %s\n", fmt.Sprintf(format, args...))
}
