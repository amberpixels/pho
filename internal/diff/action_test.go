package diff_test

import (
	"encoding/json"
	"testing"

	"pho/internal/diff"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAction_String(t *testing.T) {
	tests := []struct {
		name     string
		action   diff.Action
		expected string
	}{
		{
			name:     "diff.ActionNoop",
			action:   diff.ActionNoop,
			expected: "NOOP",
		},
		{
			name:     "diff.ActionUpdated",
			action:   diff.ActionUpdated,
			expected: "UPDATED",
		},
		{
			name:     "diff.ActionDeleted",
			action:   diff.ActionDeleted,
			expected: "DELETED",
		},
		{
			name:     "diff.ActionAdded",
			action:   diff.ActionAdded,
			expected: "ADDED",
		},
		{
			name:     "Invalid action",
			action:   diff.Action(99),
			expected: "Action(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAction_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		action   diff.Action
		expected bool
	}{
		{
			name:     "diff.ActionNoop is valid",
			action:   diff.ActionNoop,
			expected: true,
		},
		{
			name:     "diff.ActionUpdated is valid",
			action:   diff.ActionUpdated,
			expected: true,
		},
		{
			name:     "diff.ActionDeleted is valid",
			action:   diff.ActionDeleted,
			expected: true,
		},
		{
			name:     "diff.ActionAdded is valid",
			action:   diff.ActionAdded,
			expected: true,
		},
		{
			name:     "Invalid high value",
			action:   diff.Action(99),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAction_IsEffective(t *testing.T) {
	tests := []struct {
		name     string
		action   diff.Action
		expected bool
	}{
		{
			name:     "diff.ActionNoop is not effective",
			action:   diff.ActionNoop,
			expected: false,
		},
		{
			name:     "diff.ActionUpdated is effective",
			action:   diff.ActionUpdated,
			expected: true,
		},
		{
			name:     "diff.ActionDeleted is effective",
			action:   diff.ActionDeleted,
			expected: true,
		},
		{
			name:     "diff.ActionAdded is effective",
			action:   diff.ActionAdded,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action.IsEffective()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAction_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		action   diff.Action
		expected string
	}{
		{
			name:     "diff.ActionNoop marshals to NOOP",
			action:   diff.ActionNoop,
			expected: "NOOP",
		},
		{
			name:     "diff.ActionUpdated marshals to UPDATED",
			action:   diff.ActionUpdated,
			expected: "UPDATED",
		},
		{
			name:     "diff.ActionDeleted marshals to DELETED",
			action:   diff.ActionDeleted,
			expected: "DELETED",
		},
		{
			name:     "diff.ActionAdded marshals to ADDED",
			action:   diff.ActionAdded,
			expected: "ADDED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.action.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestAction_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected diff.Action
		wantErr  bool
	}{
		{
			name:     "NOOP unmarshals to diff.ActionNoop",
			input:    "NOOP",
			expected: diff.ActionNoop,
			wantErr:  false,
		},
		{
			name:     "UPDATED unmarshals to diff.ActionUpdated",
			input:    "UPDATED",
			expected: diff.ActionUpdated,
			wantErr:  false,
		},
		{
			name:     "DELETED unmarshals to diff.ActionDeleted",
			input:    "DELETED",
			expected: diff.ActionDeleted,
			wantErr:  false,
		},
		{
			name:     "ADDED unmarshals to diff.ActionAdded",
			input:    "ADDED",
			expected: diff.ActionAdded,
			wantErr:  false,
		},
		{
			name:     "Invalid string returns error",
			input:    "INVALID",
			expected: diff.ActionNoop,
			wantErr:  true,
		},
		{
			name:     "Empty string returns error",
			input:    "",
			expected: diff.ActionNoop,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action diff.Action
			err := action.UnmarshalText([]byte(tt.input))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, action)
		})
	}
}

func TestParseAction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected diff.Action
		wantErr  bool
	}{
		{
			name:     "Parse NOOP",
			input:    "NOOP",
			expected: diff.ActionNoop,
			wantErr:  false,
		},
		{
			name:     "Parse UPDATED",
			input:    "UPDATED",
			expected: diff.ActionUpdated,
			wantErr:  false,
		},
		{
			name:     "Parse DELETED",
			input:    "DELETED",
			expected: diff.ActionDeleted,
			wantErr:  false,
		},
		{
			name:     "Parse ADDED",
			input:    "ADDED",
			expected: diff.ActionAdded,
			wantErr:  false,
		},
		{
			name:     "Parse invalid",
			input:    "INVALID",
			expected: diff.ActionNoop,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := diff.ParseAction(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAction_JSONMarshaling(t *testing.T) {
	// Test JSON marshaling/unmarshaling through the MarshalText/UnmarshalText interface
	tests := []struct {
		name   string
		action diff.Action
	}{
		{
			name:   "diff.ActionNoop",
			action: diff.ActionNoop,
		},
		{
			name:   "diff.ActionUpdated",
			action: diff.ActionUpdated,
		},
		{
			name:   "diff.ActionDeleted",
			action: diff.ActionDeleted,
		},
		{
			name:   "diff.ActionAdded",
			action: diff.ActionAdded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.action)
			require.NoError(t, err)

			// Unmarshal from JSON
			var unmarshaled diff.Action
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, tt.action, unmarshaled)
		})
	}
}

func TestAction_EnumValues(t *testing.T) {
	// Test that enum values are sequential starting from 0
	assert.Equal(t, diff.ActionNoop, diff.Action(0))
	assert.Equal(t, diff.ActionUpdated, diff.Action(1))
	assert.Equal(t, diff.ActionDeleted, diff.Action(2))
	assert.Equal(t, diff.ActionAdded, diff.Action(3))
}
