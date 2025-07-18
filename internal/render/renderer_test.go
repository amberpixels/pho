package render_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"pho/internal/render"

	"go.mongodb.org/mongo-driver/bson"
)

// mockCursor implements the Cursor interface for testing.
type mockCursor struct {
	docs    []bson.M
	current int
}

func newMockCursor(docs []bson.M) *mockCursor {
	return &mockCursor{docs: docs, current: -1}
}

func (c *mockCursor) Next(_ context.Context) bool {
	c.current++
	return c.current < len(c.docs)
}

func (c *mockCursor) Decode(v any) error {
	if c.current < 0 || c.current >= len(c.docs) {
		return nil
	}

	// Simple decode implementation for testing
	if target, ok := v.(*bson.M); ok {
		*target = c.docs[c.current]
	}
	return nil
}

func TestNewRenderer(t *testing.T) {
	tests := []struct {
		name     string
		options  []render.Option
		expected *render.Configuration
	}{
		{
			name:    "no options",
			options: nil,
			expected: &render.Configuration{
				ShowLineNumbers: false,
				AsValidJSON:     false,
				ExtJSONMode:     "",
				CompactJSON:     false,
				MinimizedJSON:   false,
				IgnoreFailures:  false,
			},
		},
		{
			name:    "with options",
			options: []render.Option{render.WithShowLineNumbers(true), render.WithAsValidJSON(true)},
			expected: &render.Configuration{
				ShowLineNumbers: true,
				AsValidJSON:     true,
				ExtJSONMode:     "",
				CompactJSON:     false,
				MinimizedJSON:   false,
				IgnoreFailures:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := render.NewRenderer(tt.options...)

			if renderer == nil {
				t.Fatal("render.NewRenderer() returned nil")
			}

			config := renderer.GetConfiguration()
			if config == nil {
				t.Fatal("GetConfiguration() returned nil")
			}

			if config.ShowLineNumbers != tt.expected.ShowLineNumbers {
				t.Errorf("ShowLineNumbers = %v, want %v", config.ShowLineNumbers, tt.expected.ShowLineNumbers)
			}
			if config.AsValidJSON != tt.expected.AsValidJSON {
				t.Errorf("AsValidJSON = %v, want %v", config.AsValidJSON, tt.expected.AsValidJSON)
			}
		})
	}
}

func TestRenderer_GetConfiguration(t *testing.T) {
	renderer := render.NewRenderer(render.WithShowLineNumbers(true))
	config := renderer.GetConfiguration()

	if config == nil {
		t.Fatal("GetConfiguration() returned nil")
	}

	if !config.ShowLineNumbers {
		t.Error("Expected ShowLineNumbers to be true")
	}
}

func TestRenderer_FormatLineNumber(t *testing.T) {
	tests := []struct {
		name        string
		options     []render.Option
		lineNumber  int
		expected    string
		shouldBeNil bool
	}{
		{
			name:        "show line numbers enabled",
			options:     []render.Option{render.WithShowLineNumbers(true)},
			lineNumber:  5,
			expected:    "/* 5 */\n",
			shouldBeNil: false,
		},
		{
			name:        "show line numbers disabled",
			options:     []render.Option{render.WithShowLineNumbers(false)},
			lineNumber:  5,
			shouldBeNil: true,
		},
		{
			name:        "valid JSON mode disables line numbers",
			options:     []render.Option{render.WithShowLineNumbers(true), render.WithAsValidJSON(true)},
			lineNumber:  5,
			shouldBeNil: true,
		},
		{
			name:        "minimized JSON disables line numbers",
			options:     []render.Option{render.WithShowLineNumbers(true), render.WithMinimizedJSON(true)},
			lineNumber:  5,
			shouldBeNil: true,
		},
		{
			name:        "line number zero",
			options:     []render.Option{render.WithShowLineNumbers(true)},
			lineNumber:  0,
			expected:    "/* 0 */\n",
			shouldBeNil: false,
		},
		{
			name:        "negative line number",
			options:     []render.Option{render.WithShowLineNumbers(true)},
			lineNumber:  -1,
			expected:    "/* -1 */\n",
			shouldBeNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := render.NewRenderer(tt.options...)
			result := renderer.FormatLineNumber(tt.lineNumber)

			if tt.shouldBeNil {
				if result != nil {
					t.Errorf("FormatLineNumber() = %v, want nil", string(result))
				}
			} else {
				if result == nil {
					t.Error("FormatLineNumber() returned nil, want non-nil")
					return
				}
				if string(result) != tt.expected {
					t.Errorf("FormatLineNumber() = %v, want %v", string(result), tt.expected)
				}
			}
		})
	}
}

func TestRenderer_FormatResult(t *testing.T) {
	tests := []struct {
		name        string
		options     []render.Option
		input       bson.M
		wantErr     bool
		wantContain string
	}{
		{
			name: "canonical non-compact",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Canonical),
				render.WithCompactJSON(false),
			},
			input:       bson.M{"name": "test"},
			wantErr:     false,
			wantContain: "name",
		},
		{
			name: "canonical compact",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Canonical),
				render.WithCompactJSON(true),
			},
			input:       bson.M{"name": "test"},
			wantErr:     false,
			wantContain: "name",
		},
		{
			name: "relaxed non-compact",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
				render.WithCompactJSON(false),
			},
			input:       bson.M{"name": "test"},
			wantErr:     false,
			wantContain: "name",
		},
		{
			name: "relaxed compact",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
				render.WithCompactJSON(true),
			},
			input:       bson.M{"name": "test"},
			wantErr:     false,
			wantContain: "name",
		},
		{
			name:        "shell mode",
			options:     []render.Option{render.WithExtJSONMode(render.ExtJSONModes.Shell)},
			input:       bson.M{"name": "test"},
			wantErr:     false,
			wantContain: "name",
		},
		{
			name: "with valid JSON flag",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
				render.WithAsValidJSON(true),
			},
			input:       bson.M{"name": "test"},
			wantErr:     false,
			wantContain: ",", // Should append comma for valid JSON
		},
		{
			name: "minimized JSON",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
				render.WithMinimizedJSON(true),
			},
			input:       bson.M{"name": "test"},
			wantErr:     false,
			wantContain: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := render.NewRenderer(tt.options...)
			result, err := renderer.FormatResult(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("FormatResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(result) == 0 {
					t.Error("FormatResult() returned empty result")
					return
				}

				if tt.wantContain != "" && !strings.Contains(string(result), tt.wantContain) {
					t.Errorf("FormatResult() = %v, want to contain %v", string(result), tt.wantContain)
				}
			}
		})
	}
}

func TestRenderer_FormatResult_IgnoreFailures(t *testing.T) {
	// Test Shell mode with IgnoreFailures enabled - should work now that Shell mode is implemented
	renderer := render.NewRenderer(
		render.WithExtJSONMode(render.ExtJSONModes.Shell),
		render.WithIgnoreFailures(true),
	)

	result, err := renderer.FormatResult(bson.M{"name": "test"})

	// Should not return error and should have result since Shell mode now works
	if err != nil {
		t.Errorf("FormatResult() with Shell mode should not return error, got %v", err)
	}

	if result == nil {
		t.Error("FormatResult() with Shell mode should return result, got nil")
	}

	// Verify it contains the expected field
	if !strings.Contains(string(result), "name") {
		t.Errorf("FormatResult() result should contain 'name', got %v", string(result))
	}
}

func TestRenderer_Format(t *testing.T) {
	tests := []struct {
		name    string
		options []render.Option
		docs    []bson.M
		wantErr bool
		checkFn func(string) bool
	}{
		{
			name:    "empty cursor",
			options: []render.Option{render.WithExtJSONMode(render.ExtJSONModes.Relaxed)},
			docs:    []bson.M{},
			wantErr: false,
			checkFn: func(output string) bool { return output == "" },
		},
		{
			name:    "single document",
			options: []render.Option{render.WithExtJSONMode(render.ExtJSONModes.Relaxed)},
			docs:    []bson.M{{"name": "test"}},
			wantErr: false,
			checkFn: func(output string) bool { return strings.Contains(output, "name") },
		},
		{
			name:    "multiple documents",
			options: []render.Option{render.WithExtJSONMode(render.ExtJSONModes.Relaxed)},
			docs:    []bson.M{{"name": "test1"}, {"name": "test2"}},
			wantErr: false,
			checkFn: func(output string) bool {
				return strings.Contains(output, "test1") && strings.Contains(output, "test2")
			},
		},
		{
			name: "with line numbers",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
				render.WithShowLineNumbers(true),
			},
			docs:    []bson.M{{"name": "test"}},
			wantErr: false,
			checkFn: func(output string) bool {
				return strings.Contains(output, "/* 0 */") && strings.Contains(output, "name")
			},
		},
		{
			name:    "shell mode works",
			options: []render.Option{render.WithExtJSONMode(render.ExtJSONModes.Shell)},
			docs:    []bson.M{{"name": "test"}},
			wantErr: false,
			checkFn: func(output string) bool { return strings.Contains(output, "name") },
		},
		{
			name: "shell mode with ignore failures",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Shell),
				render.WithIgnoreFailures(true),
			},
			docs:    []bson.M{{"name": "test"}},
			wantErr: false,
			checkFn: func(output string) bool { return strings.Contains(output, "name") }, // Should contain the document
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := render.NewRenderer(tt.options...)
			cursor := newMockCursor(tt.docs)
			ctx := context.Background()

			var buf bytes.Buffer
			err := renderer.Format(ctx, cursor, &buf)

			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFn != nil {
				output := buf.String()
				if !tt.checkFn(output) {
					t.Errorf("Format() output check failed: %v", output)
				}
			}
		})
	}
}

// errorCursor is a cursor that always returns an error on Decode.
type errorCursor struct {
	callCount int
}

func (c *errorCursor) Next(_ context.Context) bool {
	c.callCount++
	return c.callCount <= 1 // Return true once to trigger Decode
}

func (c *errorCursor) Decode(_ any) error {
	return errors.New("decode error")
}

func TestRenderer_Format_DecodeError(t *testing.T) {
	tests := []struct {
		name          string
		options       []render.Option
		expectError   bool
		errorContains string
	}{
		{
			name:          "decode error without ignore failures",
			options:       []render.Option{render.WithExtJSONMode(render.ExtJSONModes.Relaxed)},
			expectError:   true,
			errorContains: "failed to decode line",
		},
		{
			name: "decode error with ignore failures",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
				render.WithIgnoreFailures(true),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := render.NewRenderer(tt.options...)
			cursor := &errorCursor{}
			ctx := context.Background()

			var buf bytes.Buffer
			err := renderer.Format(ctx, cursor, &buf)

			if (err != nil) != tt.expectError {
				t.Errorf("Format() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectError && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Format() error = %v, want error containing %v", err, tt.errorContains)
				}
			}
		})
	}
}

// writeErrorWriter is a writer that always returns an error.
type writeErrorWriter struct{}

func (w *writeErrorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}

func TestRenderer_Format_WriteError(t *testing.T) {
	tests := []struct {
		name          string
		options       []render.Option
		expectError   bool
		errorContains string
	}{
		{
			name:          "write error without ignore failures",
			options:       []render.Option{render.WithExtJSONMode(render.ExtJSONModes.Relaxed)},
			expectError:   true,
			errorContains: "failed to write line",
		},
		{
			name: "write error with ignore failures",
			options: []render.Option{
				render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
				render.WithIgnoreFailures(true),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := render.NewRenderer(tt.options...)
			cursor := newMockCursor([]bson.M{{"name": "test"}})
			ctx := context.Background()

			writer := &writeErrorWriter{}
			err := renderer.Format(ctx, cursor, writer)

			if (err != nil) != tt.expectError {
				t.Errorf("Format() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectError && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Format() error = %v, want error containing %v", err, tt.errorContains)
				}
			}
		})
	}
}

func TestRenderer_Format_ContextCancellation(_ *testing.T) {
	renderer := render.NewRenderer(render.WithExtJSONMode(render.ExtJSONModes.Relaxed))
	cursor := newMockCursor([]bson.M{{"name": "test"}})

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	err := renderer.Format(ctx, cursor, &buf)
	_ = err
	// TODO: eventually check err
}

func TestExtJSONMode_TypeSafety(t *testing.T) {
	// Test that render.ExtJSONMode is a string type
	var mode render.ExtJSONMode = "custom"
	if string(mode) != "custom" {
		t.Errorf("render.ExtJSONMode should be string-based, got %T", mode)
	}

	// Test assignment from constants
	mode = render.ExtJSONModes.Canonical
	if mode != "canonical" {
		t.Errorf("render.ExtJSONModes.Canonical = %v, want canonical", mode)
	}
}

func TestRenderer_comprehensive(t *testing.T) {
	// Test a comprehensive configuration
	renderer := render.NewRenderer(
		render.WithShowLineNumbers(true),
		render.WithAsValidJSON(true),
		render.WithExtJSONMode(render.ExtJSONModes.Canonical),
		render.WithCompactJSON(false),
		render.WithMinimizedJSON(false),
		render.WithIgnoreFailures(false),
	)

	// Verify configuration
	config := renderer.GetConfiguration()
	if !config.ShowLineNumbers {
		t.Error("Expected ShowLineNumbers to be true")
	}
	if !config.AsValidJSON {
		t.Error("Expected AsValidJSON to be true")
	}
	if config.ExtJSONMode != render.ExtJSONModes.Canonical {
		t.Errorf("Expected render.ExtJSONMode to be %v, got %v", render.ExtJSONModes.Canonical, config.ExtJSONMode)
	}

	// Test line number formatting (should be nil due to AsValidJSON)
	lineNumbers := renderer.FormatLineNumber(1)
	if lineNumbers != nil {
		t.Error("Expected line numbers to be disabled when AsValidJSON is true")
	}

	// Test result formatting
	result, err := renderer.FormatResult(bson.M{"test": "value"})
	if err != nil {
		t.Errorf("FormatResult() unexpected error: %v", err)
	}

	if !strings.Contains(string(result), ",") {
		t.Error("Expected comma for valid JSON format")
	}
}
