package hashing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewIdentifierValue(t *testing.T) {
	tests := []struct {
		name    string
		value   any
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
						assert.Fail(t, "NewIdentifierValue() unexpected panic", r)
					}
				}
			}()

			id := NewIdentifierValue(tt.value)

			if tt.wantErr {
				assert.Fail(t, "NewIdentifierValue() expected panic, got success")
				return
			}

			assert.NotNil(t, id)
			assert.Equal(t, tt.value, id.Value)
		})
	}
}

func TestIdentifierValue_String(t *testing.T) {
	tests := []struct {
		name     string
		value    any
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

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseIdentifierValue(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue any
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
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)

			// Compare values based on type
			switch expectedVal := tt.wantValue.(type) {
			case primitive.ObjectID:
				if oid, ok := result.Value.(primitive.ObjectID); ok {
					assert.Equal(t, expectedVal, oid)
				} else {
					assert.IsType(t, primitive.ObjectID{}, result.Value)
				}
			case string:
				if str, ok := result.Value.(string); ok {
					assert.Equal(t, expectedVal, str)
				} else {
					assert.IsType(t, "", result.Value)
				}
			}
		})
	}
}

func TestIdentifierValue_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value any
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
			require.NoError(t, err)

			// Compare values
			switch originalVal := tt.value.(type) {
			case primitive.ObjectID:
				if parsedOID, ok := parsed.Value.(primitive.ObjectID); ok {
					assert.Equal(t, originalVal, parsedOID)
				} else {
					assert.IsType(t, primitive.ObjectID{}, parsed.Value)
				}
			case string:
				// Note: string values get converted to ObjectID hex if they're valid hex
				// So we compare the string representation instead
				assert.Equal(t, original.String(), parsed.String())
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
