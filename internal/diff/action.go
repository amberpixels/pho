package diff

import "fmt"

// Action represents the type of change applied to a document
type Action uint8

// Action constants using iota for Go-idiomatic enum pattern
const (
	ActionNoop Action = iota
	ActionUpdated
	ActionDeleted
	ActionAdded
)

// String returns the string representation of the Action
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

// IsValid returns true if the Action value is valid
func (a Action) IsValid() bool {
	return a <= ActionAdded
}

// IsEffective returns true if the Action represents an actual change
func (a Action) IsEffective() bool {
	return a != ActionNoop
}

// MarshalText implements encoding.TextMarshaler for JSON/YAML serialization
func (a Action) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/YAML deserialization
func (a *Action) UnmarshalText(text []byte) error {
	switch string(text) {
	case "NOOP":
		*a = ActionNoop
	case "UPDATED":
		*a = ActionUpdated
	case "DELETED":
		*a = ActionDeleted
	case "ADDED":
		*a = ActionAdded
	default:
		return fmt.Errorf("invalid action: %s", string(text))
	}
	return nil
}

// ParseAction parses a string into an Action
func ParseAction(s string) (Action, error) {
	var a Action
	err := a.UnmarshalText([]byte(s))
	return a, err
}

// ActionsDict provides backward compatibility and convenience access
// Deprecated: Use Action constants directly (ActionNoop, ActionUpdated, etc.)
var ActionsDict = struct {
	Noop    Action
	Updated Action
	Deleted Action
	Added   Action
}{
	Noop:    ActionNoop,
	Updated: ActionUpdated,
	Deleted: ActionDeleted,
	Added:   ActionAdded,
}
