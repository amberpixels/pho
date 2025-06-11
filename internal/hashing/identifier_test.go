package hashing

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewIdentifierValue(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "string identifier",
			value:   "test-string-id",
			wantErr: false,
		},
		{
			name:    "ObjectID identifier",
			value:   primitive.NewObjectID(),
			wantErr: false,
		},
		{
			name:    "invalid type",
			value:   123, // int is not supported
			wantErr: true,
		},
		{
			name:    "nil value",
			value:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantErr {
						t.Errorf("NewIdentifierValue() unexpected panic: %v", r)
					}
				}
			}()
			
			id := NewIdentifierValue(tt.value)
			
			if tt.wantErr {
				t.Errorf("NewIdentifierValue() expected panic, got success")
				return
			}
			
			if id == nil {
				t.Errorf("NewIdentifierValue() returned nil")
				return
			}
			
			if id.Value != tt.value {
				t.Errorf("NewIdentifierValue() value mismatch: got %v, want %v", id.Value, tt.value)
			}
		})
	}
}

func TestIdentifierValue_String(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string value",
			value:    "test-id",
			expected: "test-id",
		},
		{
			name:     "ObjectID value",
			value:    mustObjectIDFromHex("507f1f77bcf86cd799439011"),
			expected: "ObjectID(\"507f1f77bcf86cd799439011\")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := &IdentifierValue{Value: tt.value}
			result := id.String()
			
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseIdentifierValue(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue interface{}
		wantErr   bool
	}{
		{
			name:      "ObjectID format",
			input:     "ObjectID(\"507f1f77bcf86cd799439011\")",
			wantValue: mustObjectIDFromHex("507f1f77bcf86cd799439011"),
			wantErr:   false,
		},
		{
			name:      "string as ObjectID hex",
			input:     "507f1f77bcf86cd799439012",
			wantValue: "507f1f77bcf86cd799439012",
			wantErr:   false,
		},
		{
			name:      "invalid ObjectID format",
			input:     "ObjectID(\"invalid-hex\")",
			wantValue: nil,
			wantErr:   true,
		},
		{
			name:      "invalid hex string",
			input:     "not-a-valid-hex",
			wantValue: nil,
			wantErr:   true,
		},
		{
			name:      "empty string",
			input:     "",
			wantValue: nil,
			wantErr:   true,
		},
		{
			name:      "malformed ObjectID",
			input:     "ObjectID(507f1f77bcf86cd799439011)",
			wantValue: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseIdentifierValue(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseIdentifierValue() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseIdentifierValue() unexpected error: %v", err)
				return
			}
			
			if result == nil {
				t.Errorf("ParseIdentifierValue() returned nil result")
				return
			}
			
			// Compare values based on type
			switch expectedVal := tt.wantValue.(type) {
			case primitive.ObjectID:
				if oid, ok := result.Value.(primitive.ObjectID); ok {
					if oid != expectedVal {
						t.Errorf("ParseIdentifierValue() ObjectID mismatch: got %v, want %v", oid, expectedVal)
					}
				} else {
					t.Errorf("ParseIdentifierValue() expected ObjectID, got %T", result.Value)
				}
			case string:
				if str, ok := result.Value.(string); ok {
					if str != expectedVal {
						t.Errorf("ParseIdentifierValue() string mismatch: got %v, want %v", str, expectedVal)
					}
				} else {
					t.Errorf("ParseIdentifierValue() expected string, got %T", result.Value)
				}
			}
		})
	}
}

func TestIdentifierValue_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "string round trip",
			value: "507f1f77bcf86cd799439013", // Use valid hex string
		},
		{
			name:  "ObjectID round trip",
			value: mustObjectIDFromHex("507f1f77bcf86cd799439011"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create identifier value
			original := &IdentifierValue{Value: tt.value}
			
			// Convert to string
			str := original.String()
			
			// Parse back
			parsed, err := ParseIdentifierValue(str)
			if err != nil {
				t.Fatalf("ParseIdentifierValue() failed: %v", err)
			}
			
			// Compare values
			switch originalVal := tt.value.(type) {
			case primitive.ObjectID:
				if parsedOID, ok := parsed.Value.(primitive.ObjectID); ok {
					if parsedOID != originalVal {
						t.Errorf("Round trip failed for ObjectID: got %v, want %v", parsedOID, originalVal)
					}
				} else {
					t.Errorf("Round trip failed: expected ObjectID, got %T", parsed.Value)
				}
			case string:
				// Note: string values get converted to ObjectID hex if they're valid hex
				// So we compare the string representation instead
				if parsed.String() != original.String() {
					t.Errorf("Round trip failed for string: got %v, want %v", parsed.String(), original.String())
				}
			}
		})
	}
}

// Helper function to create ObjectID from hex string, panics on error
func mustObjectIDFromHex(hex string) primitive.ObjectID {
	oid, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		panic(err)
	}
	return oid
}