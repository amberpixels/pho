package diff

import (
	"encoding/json"
	"testing"
)

func TestAction_String(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected string
	}{
		{
			name:     "ActionNoop",
			action:   ActionNoop,
			expected: "NOOP",
		},
		{
			name:     "ActionUpdated",
			action:   ActionUpdated,
			expected: "UPDATED",
		},
		{
			name:     "ActionDeleted",
			action:   ActionDeleted,
			expected: "DELETED",
		},
		{
			name:     "ActionAdded",
			action:   ActionAdded,
			expected: "ADDED",
		},
		{
			name:     "Invalid action",
			action:   Action(99),
			expected: "Action(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action.String()
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAction_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected bool
	}{
		{
			name:     "ActionNoop is valid",
			action:   ActionNoop,
			expected: true,
		},
		{
			name:     "ActionUpdated is valid",
			action:   ActionUpdated,
			expected: true,
		},
		{
			name:     "ActionDeleted is valid",
			action:   ActionDeleted,
			expected: true,
		},
		{
			name:     "ActionAdded is valid",
			action:   ActionAdded,
			expected: true,
		},
		{
			name:     "Invalid high value",
			action:   Action(99),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAction_IsEffective(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected bool
	}{
		{
			name:     "ActionNoop is not effective",
			action:   ActionNoop,
			expected: false,
		},
		{
			name:     "ActionUpdated is effective",
			action:   ActionUpdated,
			expected: true,
		},
		{
			name:     "ActionDeleted is effective",
			action:   ActionDeleted,
			expected: true,
		},
		{
			name:     "ActionAdded is effective",
			action:   ActionAdded,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.action.IsEffective()
			if result != tt.expected {
				t.Errorf("IsEffective() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAction_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected string
	}{
		{
			name:     "ActionNoop marshals to NOOP",
			action:   ActionNoop,
			expected: "NOOP",
		},
		{
			name:     "ActionUpdated marshals to UPDATED",
			action:   ActionUpdated,
			expected: "UPDATED",
		},
		{
			name:     "ActionDeleted marshals to DELETED",
			action:   ActionDeleted,
			expected: "DELETED",
		},
		{
			name:     "ActionAdded marshals to ADDED",
			action:   ActionAdded,
			expected: "ADDED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.action.MarshalText()
			if err != nil {
				t.Errorf("MarshalText() error = %v", err)
				return
			}
			if string(result) != tt.expected {
				t.Errorf("MarshalText() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

func TestAction_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Action
		wantErr  bool
	}{
		{
			name:     "NOOP unmarshals to ActionNoop",
			input:    "NOOP",
			expected: ActionNoop,
			wantErr:  false,
		},
		{
			name:     "UPDATED unmarshals to ActionUpdated",
			input:    "UPDATED",
			expected: ActionUpdated,
			wantErr:  false,
		},
		{
			name:     "DELETED unmarshals to ActionDeleted",
			input:    "DELETED",
			expected: ActionDeleted,
			wantErr:  false,
		},
		{
			name:     "ADDED unmarshals to ActionAdded",
			input:    "ADDED",
			expected: ActionAdded,
			wantErr:  false,
		},
		{
			name:     "Invalid string returns error",
			input:    "INVALID",
			expected: ActionNoop,
			wantErr:  true,
		},
		{
			name:     "Empty string returns error",
			input:    "",
			expected: ActionNoop,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action Action
			err := action.UnmarshalText([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Errorf("UnmarshalText() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("UnmarshalText() unexpected error: %v", err)
				return
			}

			if action != tt.expected {
				t.Errorf("UnmarshalText() = %v, want %v", action, tt.expected)
			}
		})
	}
}

func TestParseAction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Action
		wantErr  bool
	}{
		{
			name:     "Parse NOOP",
			input:    "NOOP",
			expected: ActionNoop,
			wantErr:  false,
		},
		{
			name:     "Parse UPDATED",
			input:    "UPDATED",
			expected: ActionUpdated,
			wantErr:  false,
		},
		{
			name:     "Parse DELETED",
			input:    "DELETED",
			expected: ActionDeleted,
			wantErr:  false,
		},
		{
			name:     "Parse ADDED",
			input:    "ADDED",
			expected: ActionAdded,
			wantErr:  false,
		},
		{
			name:     "Parse invalid",
			input:    "INVALID",
			expected: ActionNoop,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseAction(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseAction() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseAction() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("ParseAction() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAction_JSONMarshaling(t *testing.T) {
	// Test JSON marshaling/unmarshaling through the MarshalText/UnmarshalText interface
	tests := []struct {
		name   string
		action Action
	}{
		{
			name:   "ActionNoop",
			action: ActionNoop,
		},
		{
			name:   "ActionUpdated",
			action: ActionUpdated,
		},
		{
			name:   "ActionDeleted",
			action: ActionDeleted,
		},
		{
			name:   "ActionAdded",
			action: ActionAdded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.action)
			if err != nil {
				t.Errorf("json.Marshal() error = %v", err)
				return
			}

			// Unmarshal from JSON
			var unmarshaled Action
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("json.Unmarshal() error = %v", err)
				return
			}

			// Compare
			if unmarshaled != tt.action {
				t.Errorf("JSON round-trip failed: got %v, want %v", unmarshaled, tt.action)
			}
		})
	}
}

func TestActionsDict_BackwardCompatibility(t *testing.T) {
	// Test that ActionsDict still works for backward compatibility
	tests := []struct {
		name string
		old  Action
		new  Action
	}{
		{
			name: "Noop compatibility",
			old:  ActionsDict.Noop,
			new:  ActionNoop,
		},
		{
			name: "Updated compatibility",
			old:  ActionsDict.Updated,
			new:  ActionUpdated,
		},
		{
			name: "Deleted compatibility",
			old:  ActionsDict.Deleted,
			new:  ActionDeleted,
		},
		{
			name: "Added compatibility",
			old:  ActionsDict.Added,
			new:  ActionAdded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.old != tt.new {
				t.Errorf("Backward compatibility broken: %v != %v", tt.old, tt.new)
			}
		})
	}
}

func TestAction_EnumValues(t *testing.T) {
	// Test that enum values are sequential starting from 0
	if ActionNoop != 0 {
		t.Errorf("ActionNoop should be 0, got %d", ActionNoop)
	}
	if ActionUpdated != 1 {
		t.Errorf("ActionUpdated should be 1, got %d", ActionUpdated)
	}
	if ActionDeleted != 2 {
		t.Errorf("ActionDeleted should be 2, got %d", ActionDeleted)
	}
	if ActionAdded != 3 {
		t.Errorf("ActionAdded should be 3, got %d", ActionAdded)
	}
}
