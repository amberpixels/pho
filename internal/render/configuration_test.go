package render

import (
	"testing"
)

func TestNewConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		expected *Configuration
	}{
		{
			name:    "no options",
			options: nil,
			expected: &Configuration{
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
			options: []Option{WithShowLineNumbers(true)},
			expected: &Configuration{
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
			options: []Option{WithAsValidJSON(true)},
			expected: &Configuration{
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
			options: []Option{WithExtJSONMode(ExtJSONModes.Canonical)},
			expected: &Configuration{
				ShowLineNumbers: false,
				AsValidJSON:     false,
				ExtJSONMode:     ExtJSONModes.Canonical,
				CompactJSON:     false,
				MinimizedJSON:   false,
				IgnoreFailures:  false,
			},
		},
		{
			name:    "with compact JSON",
			options: []Option{WithCompactJSON(true)},
			expected: &Configuration{
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
			options: []Option{WithMinimizedJSON(true)},
			expected: &Configuration{
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
			options: []Option{WithIgnoreFailures(true)},
			expected: &Configuration{
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
			options: []Option{
				WithShowLineNumbers(true),
				WithAsValidJSON(true),
				WithExtJSONMode(ExtJSONModes.Relaxed),
				WithCompactJSON(true),
				WithIgnoreFailures(true),
			},
			expected: &Configuration{
				ShowLineNumbers: true,
				AsValidJSON:     true,
				ExtJSONMode:     ExtJSONModes.Relaxed,
				CompactJSON:     true,
				MinimizedJSON:   false,
				IgnoreFailures:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfiguration(tt.options...)
			
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
	if ExtJSONModes.Canonical != "canonical" {
		t.Errorf("ExtJSONModes.Canonical = %v, want canonical", ExtJSONModes.Canonical)
	}
	if ExtJSONModes.Relaxed != "relaxed" {
		t.Errorf("ExtJSONModes.Relaxed = %v, want relaxed", ExtJSONModes.Relaxed)
	}
	if ExtJSONModes.Shell != "shell" {
		t.Errorf("ExtJSONModes.Shell = %v, want shell", ExtJSONModes.Shell)
	}
}

func TestConfiguration_Clone(t *testing.T) {
	original := NewConfiguration(
		WithShowLineNumbers(true),
		WithAsValidJSON(true),
		WithExtJSONMode(ExtJSONModes.Canonical),
		WithCompactJSON(true),
		WithMinimizedJSON(true),
		WithIgnoreFailures(true),
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
			config := &Configuration{}
			option := WithShowLineNumbers(tt.value)
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
			config := &Configuration{}
			option := WithAsValidJSON(tt.value)
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
		value ExtJSONMode
	}{
		{"canonical", ExtJSONModes.Canonical},
		{"relaxed", ExtJSONModes.Relaxed},
		{"shell", ExtJSONModes.Shell},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Configuration{}
			option := WithExtJSONMode(tt.value)
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
			config := &Configuration{}
			option := WithCompactJSON(tt.value)
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
			config := &Configuration{}
			option := WithMinimizedJSON(tt.value)
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
			config := &Configuration{}
			option := WithIgnoreFailures(tt.value)
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
		input    interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "Configuration struct",
			input: &Configuration{
				ShowLineNumbers: true,
				AsValidJSON:     true,
				ExtJSONMode:     ExtJSONModes.Canonical,
				CompactJSON:     true,
				MinimizedJSON:   false,
				IgnoreFailures:  true,
			},
			expected: &Configuration{},
			wantErr:  false,
		},
		{
			name: "simple map",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
			expected: &map[string]interface{}{},
			wantErr:  false,
		},
		{
			name: "nil input",
			input: (*Configuration)(nil),
			expected: &Configuration{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RoundTripJSON(tt.input, tt.expected)
			
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
	output := &Configuration{}
	
	err := RoundTripJSON(invalidInput, output)
	if err == nil {
		t.Error("RoundTripJSON() expected error for invalid input, got nil")
	}
}

func TestRoundTripJSON_InvalidOutput(t *testing.T) {
	// Test with valid input but invalid output type
	input := map[string]interface{}{"key": "value"}
	var output string // string can't hold the map structure
	
	err := RoundTripJSON(input, &output)
	if err == nil {
		t.Error("RoundTripJSON() expected error for type mismatch, got nil")
	}
}

func TestOptions_Chaining(t *testing.T) {
	// Test that options can be chained together
	config := NewConfiguration(
		WithShowLineNumbers(true),
		WithAsValidJSON(true),
		WithExtJSONMode(ExtJSONModes.Relaxed),
		WithCompactJSON(true),
		WithMinimizedJSON(true),
		WithIgnoreFailures(true),
	)
	
	if !config.ShowLineNumbers {
		t.Error("Expected ShowLineNumbers to be true")
	}
	if !config.AsValidJSON {
		t.Error("Expected AsValidJSON to be true")
	}
	if config.ExtJSONMode != ExtJSONModes.Relaxed {
		t.Errorf("Expected ExtJSONMode to be %v, got %v", ExtJSONModes.Relaxed, config.ExtJSONMode)
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
	config := NewConfiguration(
		WithShowLineNumbers(false),
		WithShowLineNumbers(true), // Should override the false
		WithExtJSONMode(ExtJSONModes.Canonical),
		WithExtJSONMode(ExtJSONModes.Relaxed), // Should override canonical
	)
	
	if !config.ShowLineNumbers {
		t.Error("Expected ShowLineNumbers to be true (overridden)")
	}
	if config.ExtJSONMode != ExtJSONModes.Relaxed {
		t.Errorf("Expected ExtJSONMode to be %v (overridden), got %v", ExtJSONModes.Relaxed, config.ExtJSONMode)
	}
}