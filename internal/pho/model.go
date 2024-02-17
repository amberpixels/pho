package pho

import (
	"go.mongodb.org/mongo-driver/bson"
	"pho/internal/hashing"
)

type ParsedMeta struct {
	// todo:
	// db name
	// collection name
	// ExtJSON params

	// Lines are hashes per identifier.
	// Identifier here is considered to be identified_by field + identifier value
	// etc. _id:111111
	Lines map[string]*hashing.HashData
}

type Action string

var Actions = struct {
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

type Change struct {
	hash    *hashing.HashData
	action  Action
	command string // string for now
}
type Changes []*Change

func (chs Changes) Filter(f func(*Change) bool) Changes {
	filtered := make(Changes, 0)
	for _, ch := range chs {
		if f(ch) {
			filtered = append(filtered, ch)
		}
	}
	return filtered
}

func (chs Changes) Effective() Changes {
	return chs.Filter(func(ch *Change) bool { return ch.action != Actions.Noop })
}

func (chs Changes) EffectiveCount() int {
	count := 0
	for _, ch := range chs {
		if ch.action != Actions.Noop {
			count++
		}
	}
	return count
}

func NewChange(hashData *hashing.HashData, action Action, command string) *Change {
	return &Change{hash: hashData, action: action, command: command}
}

type DumpDoc bson.M

// UnmarshalJSON for now is a hack, as we hardcode the way unmarshal parameters in here
// Whole thing of DumpDoc is required, so it's properly parsed back from ExtJson into bson
func (tx *DumpDoc) UnmarshalJSON(raw []byte) error {
	return bson.UnmarshalExtJSON(raw, true, tx)
}
