package diff

import "fmt"

// Action represents the type of change applied to a document.
type Action uint8

// Action constants using iota for Go-idiomatic enum pattern.
const (
	ActionNoop Action = iota
	ActionUpdated
	ActionDeleted
	ActionAdded
)

// String returns the string representation of the Action.
func (a Action) String() string {
	switch a {
	case ActionNoop:
		return "NOOP"
	case ActionUpdated:
		return "UPDATED"
	case ActionDeleted:
		return "DELETED"
	case ActionAdded:
		return "ADDED"
	default:
		return fmt.Sprintf("Action(%d)", uint8(a))
	}
}

// IsValid returns true if the Action value is valid.
func (a Action) IsValid() bool {
	return a <= ActionAdded
}

// IsEffective returns true if the Action represents an actual change.
func (a Action) IsEffective() bool {
	return a != ActionNoop
}

// MarshalText implements encoding.TextMarshaler for JSON/YAML serialization.
func (a Action) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/YAML deserialization.
func (a *Action) UnmarshalText(text []byte) error {
	input := string(text)

	// Try each action by comparing with its String() representation
	for candidate := ActionNoop; candidate <= ActionAdded; candidate++ {
		if input == candidate.String() {
			*a = candidate
			return nil
		}
	}

	return fmt.Errorf("invalid action: %s", input)
}

// ParseAction parses a string into an Action.
func ParseAction(s string) (Action, error) {
	var a Action
	err := a.UnmarshalText([]byte(s))
	return a, err
}
