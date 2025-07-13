package render_test

import (
	"pho/internal/render"
	"testing"
)

func TestNewConfiguration(t *testing.T) {
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
			name:    "with show line numbers",
			options: []render.Option{render.WithShowLineNumbers(true)},
			expected: &render.Configuration{
				ShowLineNumbers: true,
				AsValidJSON:     false,
				ExtJSONMode:     "",
				CompactJSON:     false,
				MinimizedJSON:   false,
				IgnoreFailures:  false,
			},
		},
		{
			name:    "with valid JSON",
			options: []render.Option{render.WithAsValidJSON(true)},
			expected: &render.Configuration{
				ShowLineNumbers: false,
				AsValidJSON:     true,
				ExtJSONMode:     "",
				CompactJSON:     false,
				MinimizedJSON:   false,
				IgnoreFailures:  false,
			},
		},
		{
			name:    "with ExtJSON mode",
			options: []render.Option{render.WithExtJSONMode(render.ExtJSONModes.Canonical)},
			expected: &render.Configuration{
				ShowLineNumbers: false,
				AsValidJSON:     false,
				ExtJSONMode:     render.ExtJSONModes.Canonical,
				CompactJSON:     false,
				MinimizedJSON:   false,
				IgnoreFailures:  false,
			},
		},
		{
			name:    "with compact JSON",
			options: []render.Option{render.WithCompactJSON(true)},
			expected: &render.Configuration{
				ShowLineNumbers: false,
				AsValidJSON:     false,
				ExtJSONMode:     "",
				CompactJSON:     true,
				MinimizedJSON:   false,
				IgnoreFailures:  false,
			},
		},
		{
			name:    "with minimized JSON",
			options: []render.Option{render.WithMinimizedJSON(true)},
			expected: &render.Configuration{
				ShowLineNumbers: false,
				AsValidJSON:     false,
				ExtJSONMode:     "",
				CompactJSON:     false,
				MinimizedJSON:   true,
				IgnoreFailures:  false,
			},
		},
		{
			name:    "with ignore failures",
			options: []render.Option{render.WithIgnoreFailures(true)},
			expected: &render.Configuration{
				ShowLineNumbers: false,
				AsValidJSON:     false,
				ExtJSONMode:     "",
				CompactJSON:     false,
				MinimizedJSON:   false,
				IgnoreFailures:  true,
			},
		},
		{
			name: "with multiple options",
			options: []render.Option{
				render.WithShowLineNumbers(true),
				render.WithAsValidJSON(true),
				render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
				render.WithCompactJSON(true),
				render.WithIgnoreFailures(true),
			},
			expected: &render.Configuration{
				ShowLineNumbers: true,
				AsValidJSON:     true,
				ExtJSONMode:     render.ExtJSONModes.Relaxed,
				CompactJSON:     true,
				MinimizedJSON:   false,
				IgnoreFailures:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := render.NewConfiguration(tt.options...)

			if config.ShowLineNumbers != tt.expected.ShowLineNumbers {
				t.Errorf("ShowLineNumbers = %v, want %v", config.ShowLineNumbers, tt.expected.ShowLineNumbers)
			}
			if config.AsValidJSON != tt.expected.AsValidJSON {
				t.Errorf("AsValidJSON = %v, want %v", config.AsValidJSON, tt.expected.AsValidJSON)
			}
			if config.ExtJSONMode != tt.expected.ExtJSONMode {
				t.Errorf("ExtJSONMode = %v, want %v", config.ExtJSONMode, tt.expected.ExtJSONMode)
			}
			if config.CompactJSON != tt.expected.CompactJSON {
				t.Errorf("CompactJSON = %v, want %v", config.CompactJSON, tt.expected.CompactJSON)
			}
			if config.MinimizedJSON != tt.expected.MinimizedJSON {
				t.Errorf("MinimizedJSON = %v, want %v", config.MinimizedJSON, tt.expected.MinimizedJSON)
			}
			if config.IgnoreFailures != tt.expected.IgnoreFailures {
				t.Errorf("IgnoreFailures = %v, want %v", config.IgnoreFailures, tt.expected.IgnoreFailures)
			}
		})
	}
}

func TestExtJSONModes(t *testing.T) {
	if render.ExtJSONModes.Canonical != "canonical" {
		t.Errorf("ExtJSONModes.Canonical = %v, want canonical", render.ExtJSONModes.Canonical)
	}
	if render.ExtJSONModes.Relaxed != "relaxed" {
		t.Errorf("ExtJSONModes.Relaxed = %v, want relaxed", render.ExtJSONModes.Relaxed)
	}
	if render.ExtJSONModes.Shell != "shell" {
		t.Errorf("ExtJSONModes.Shell = %v, want shell", render.ExtJSONModes.Shell)
	}
}

func TestConfiguration_Clone(t *testing.T) {
	original := render.NewConfiguration(
		render.WithShowLineNumbers(true),
		render.WithAsValidJSON(true),
		render.WithExtJSONMode(render.ExtJSONModes.Canonical),
		render.WithCompactJSON(true),
		render.WithMinimizedJSON(true),
		render.WithIgnoreFailures(true),
	)

	cloned := original.Clone()

	// Verify clone has same values
	if cloned.ShowLineNumbers != original.ShowLineNumbers {
		t.Errorf("Clone ShowLineNumbers = %v, want %v", cloned.ShowLineNumbers, original.ShowLineNumbers)
	}
	if cloned.AsValidJSON != original.AsValidJSON {
		t.Errorf("Clone AsValidJSON = %v, want %v", cloned.AsValidJSON, original.AsValidJSON)
	}
	if cloned.ExtJSONMode != original.ExtJSONMode {
		t.Errorf("Clone ExtJSONMode = %v, want %v", cloned.ExtJSONMode, original.ExtJSONMode)
	}
	if cloned.CompactJSON != original.CompactJSON {
		t.Errorf("Clone CompactJSON = %v, want %v", cloned.CompactJSON, original.CompactJSON)
	}
	if cloned.MinimizedJSON != original.MinimizedJSON {
		t.Errorf("Clone MinimizedJSON = %v, want %v", cloned.MinimizedJSON, original.MinimizedJSON)
	}
	if cloned.IgnoreFailures != original.IgnoreFailures {
		t.Errorf("Clone IgnoreFailures = %v, want %v", cloned.IgnoreFailures, original.IgnoreFailures)
	}

	// Verify they are different objects
	if cloned == original {
		t.Error("Clone should return a different object")
	}

	// Verify modifying clone doesn't affect original
	cloned.ShowLineNumbers = !cloned.ShowLineNumbers
	if cloned.ShowLineNumbers == original.ShowLineNumbers {
		t.Error("Modifying clone should not affect original")
	}
}

func TestWithShowLineNumbers(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &render.Configuration{}
			option := render.WithShowLineNumbers(tt.value)
			option(config)

			if config.ShowLineNumbers != tt.value {
				t.Errorf("WithShowLineNumbers(%v) = %v, want %v", tt.value, config.ShowLineNumbers, tt.value)
			}
		})
	}
}

func TestWithAsValidJSON(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &render.Configuration{}
			option := render.WithAsValidJSON(tt.value)
			option(config)

			if config.AsValidJSON != tt.value {
				t.Errorf("WithAsValidJSON(%v) = %v, want %v", tt.value, config.AsValidJSON, tt.value)
			}
		})
	}
}

func TestWithExtJSONMode(t *testing.T) {
	tests := []struct {
		name  string
		value render.ExtJSONMode
	}{
		{"canonical", render.ExtJSONModes.Canonical},
		{"relaxed", render.ExtJSONModes.Relaxed},
		{"shell", render.ExtJSONModes.Shell},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &render.Configuration{}
			option := render.WithExtJSONMode(tt.value)
			option(config)

			if config.ExtJSONMode != tt.value {
				t.Errorf("WithExtJSONMode(%v) = %v, want %v", tt.value, config.ExtJSONMode, tt.value)
			}
		})
	}
}

func TestWithCompactJSON(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &render.Configuration{}
			option := render.WithCompactJSON(tt.value)
			option(config)

			if config.CompactJSON != tt.value {
				t.Errorf("WithCompactJSON(%v) = %v, want %v", tt.value, config.CompactJSON, tt.value)
			}
		})
	}
}

func TestWithMinimizedJSON(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &render.Configuration{}
			option := render.WithMinimizedJSON(tt.value)
			option(config)

			if config.MinimizedJSON != tt.value {
				t.Errorf("WithMinimizedJSON(%v) = %v, want %v", tt.value, config.MinimizedJSON, tt.value)
			}
		})
	}
}

func TestWithIgnoreFailures(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &render.Configuration{}
			option := render.WithIgnoreFailures(tt.value)
			option(config)

			if config.IgnoreFailures != tt.value {
				t.Errorf("WithIgnoreFailures(%v) = %v, want %v", tt.value, config.IgnoreFailures, tt.value)
			}
		})
	}
}

func TestRoundTripJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
		wantErr  bool
	}{
		{
			name: "Configuration struct",
			input: &render.Configuration{
				ShowLineNumbers: true,
				AsValidJSON:     true,
				ExtJSONMode:     render.ExtJSONModes.Canonical,
				CompactJSON:     true,
				MinimizedJSON:   false,
				IgnoreFailures:  true,
			},
			expected: &render.Configuration{},
			wantErr:  false,
		},
		{
			name: "simple map",
			input: map[string]any{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			expected: &map[string]any{},
			wantErr:  false,
		},
		{
			name:     "nil input",
			input:    (*render.Configuration)(nil),
			expected: &render.Configuration{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := render.RoundTripJSON(tt.input, tt.expected)

			if (err != nil) != tt.wantErr {
				t.Errorf("RoundTripJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// For successful cases, verify the output is not nil
				if tt.expected == nil {
					t.Error("RoundTripJSON() output should not be nil")
				}
			}
		})
	}
}

func TestRoundTripJSON_InvalidInput(t *testing.T) {
	// Test with invalid input that can't be marshaled to JSON
	invalidInput := func() {} // functions can't be marshaled to JSON
	output := &render.Configuration{}

	err := render.RoundTripJSON(invalidInput, output)
	if err == nil {
		t.Error("RoundTripJSON() expected error for invalid input, got nil")
	}
}

func TestRoundTripJSON_InvalidOutput(t *testing.T) {
	// Test with valid input but invalid output type
	input := map[string]any{"key": "value"}
	var output string // string can't hold the map structure

	err := render.RoundTripJSON(input, &output)
	if err == nil {
		t.Error("RoundTripJSON() expected error for type mismatch, got nil")
	}
}

func TestOptions_Chaining(t *testing.T) {
	// Test that options can be chained together
	config := render.NewConfiguration(
		render.WithShowLineNumbers(true),
		render.WithAsValidJSON(true),
		render.WithExtJSONMode(render.ExtJSONModes.Relaxed),
		render.WithCompactJSON(true),
		render.WithMinimizedJSON(true),
		render.WithIgnoreFailures(true),
	)

	if !config.ShowLineNumbers {
		t.Error("Expected ShowLineNumbers to be true")
	}
	if !config.AsValidJSON {
		t.Error("Expected AsValidJSON to be true")
	}
	if config.ExtJSONMode != render.ExtJSONModes.Relaxed {
		t.Errorf("Expected ExtJSONMode to be %v, got %v", render.ExtJSONModes.Relaxed, config.ExtJSONMode)
	}
	if !config.CompactJSON {
		t.Error("Expected CompactJSON to be true")
	}
	if !config.MinimizedJSON {
		t.Error("Expected MinimizedJSON to be true")
	}
	if !config.IgnoreFailures {
		t.Error("Expected IgnoreFailures to be true")
	}
}

func TestOptions_Override(t *testing.T) {
	// Test that later options override earlier ones
	config := render.NewConfiguration(
		render.WithShowLineNumbers(false),
		render.WithShowLineNumbers(true), // Should override the false
		render.WithExtJSONMode(render.ExtJSONModes.Canonical),
		render.WithExtJSONMode(render.ExtJSONModes.Relaxed), // Should override canonical
	)

	if !config.ShowLineNumbers {
		t.Error("Expected ShowLineNumbers to be true (overridden)")
	}
	if config.ExtJSONMode != render.ExtJSONModes.Relaxed {
		t.Errorf("Expected ExtJSONMode to be %v (overridden), got %v", render.ExtJSONModes.Relaxed, config.ExtJSONMode)
	}
}
