package diff

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"pho/internal/hashing"
)

// Change holds a document (represented by its hash) with action that was applied.
// Having a document (bson.M) + a Change allows to create a changed version of the doc.
type Change struct {
	action Action

	hash *hashing.HashData
}

func NewChange(hashData *hashing.HashData, action Action) *Change {
	return &Change{hash: hashData, action: action}
}

type Changes []*Change

// Filter returns a filtered list of changes (by a given filter func(
func (chs Changes) Filter(f func(*Change) bool) Changes {
	filtered := make(Changes, 0)
	for _, ch := range chs {
		if f(ch) {
			filtered = append(filtered, ch)
		}
	}
	return filtered
}

// Effective is an alias for Filter(action!=noop)
func (chs Changes) Effective() Changes {
	return chs.Filter(func(ch *Change) bool { return ch.action != ActionsDict.Noop })
}

// EffectiveCount returns number of effective changes
func (chs Changes) EffectiveCount() int {
	count := 0
	for _, ch := range chs {
		if ch.action != ActionsDict.Noop {
			count++
		}
	}
	return count
}

// ChangesPack stores a list of changes
// With the list of shell commands (corresponding to re-create doc with the changes)
// Relation between changes vs shellCommands is made via slice index
// So len(changes) MUST equal len(shellCommands) and their positions match
type ChangesPack struct {
	changes       Changes
	shellCommands []string
}

func NewChangesPack() *ChangesPack {
	return &ChangesPack{make(Changes, 0), make([]string, 0)}
}

func (cp *ChangesPack) Add(c *Change, shellCmd string) *ChangesPack {
	cp.changes = append(cp.changes, c)
	cp.shellCommands = append(cp.shellCommands, shellCmd)
	return cp
}

func (cp *ChangesPack) Changes() Changes {
	return cp.changes
}

// CalculateChanges calculates changes that represent difference between
// given `source` hashed lines and `destination` list of current versions of documents
func CalculateChanges(source map[string]*hashing.HashData, destination []bson.M) (*ChangesPack, error) {

	// to avoid multiple iterations across changes
	// we want to compile commands during calculation of changes
	// TODO: solve params here, For now they are hardcoded
	cmdBuilder := NewCmdBuilder("mydb", "samples")

	// n stands for length of destination slice
	n := len(destination)
	shellCommands := make([]string, n)
	changes := make(Changes, n)

	// hashmap for documents that were processed
	idsLUT := make(map[string]struct{})
	for i, obj := range destination {
		hashData, err := hashing.Hash(obj)
		if err != nil {
			return nil, fmt.Errorf("corrupted obj[%d] could not hash: %w", i, err)
		}
		checksumAfter := hashData.GetChecksum()

		//slog.Log(context.Background(), slog.LevelInfo, "AFTER "+hashData.String())

		id := hashData.GetIdentifier()
		idsLUT[id] = struct{}{}

		hashDataBefore, ok := source[id]
		if !ok {
			// not found in source, so it was added
			changes[i] = NewChange(hashData, ActionsDict.Added)
			shellCommands[i] = "todo added"
			continue
		}

		//slog.Log(context.Background(), slog.LevelInfo, "BEFORE "+checksumBefore)

		if hashDataBefore.GetChecksum() == checksumAfter {
			changes[i] = NewChange(hashData, ActionsDict.Noop)
			shellCommands[i] = "todo: noop"
			continue
		}

		c := NewChange(hashData, ActionsDict.Updated)
		changes[i] = c

		cmd, err := cmdBuilder.BuildShellCommand(c, obj)
		if err != nil {
			return nil, fmt.Errorf("could not build shel command: %w", err)
		}
		shellCommands[i] = cmd
	}

	// Going the other way source=>destination
	// to find documents that were deleted
	// Note: order of deletion documents will not be respected
	for existedBeforeIdentifier, hashData := range source {
		if _, ok := idsLUT[existedBeforeIdentifier]; ok {
			continue
		}

		changes = append(changes, NewChange(hashData, ActionsDict.Deleted))
		shellCommands = append(shellCommands, "todo: deleted")
	}

	return &ChangesPack{changes, shellCommands}, nil
}
