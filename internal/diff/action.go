package diff

// todo: apply a richer enum declaration

// Action that was applied to the given doc
// Note: never create values on the fly:
//
//	Always use ActionsDict struct instead.
type Action string

// ActionsDict holds all available Action values
var ActionsDict = struct {
	Noop    Action
	Updated Action
	Deleted Action
	Added   Action
}{
	Noop:    "NOOP",
	Updated: "UPDATED",

	// Not considered to be supported right this minute:
	Deleted: "DELETED",
	Added:   "ADDED",
}

// IsValid checks if given action value is a valid value: that it's a part of ActionsDict
func IsValid(a Action) bool {
	switch a {
	case ActionsDict.Noop, ActionsDict.Updated,
		ActionsDict.Added, ActionsDict.Deleted:
		return true
	}

	return false
}
