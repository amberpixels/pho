package diff

// TODO: apply a richer enum declaration

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
