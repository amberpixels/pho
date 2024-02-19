package diff

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
