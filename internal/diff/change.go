package diff

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"pho/internal/hashing"
)

// ParsedMeta stores hashed lines and other meta
// TODO: should not be part of diff package
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

type Change struct {
	action Action

	hash *hashing.HashData
}

func NewChange(hashData *hashing.HashData, action Action) *Change {
	return &Change{hash: hashData, action: action}
}

type Changes []*Change

type ChangesPack struct {
	// len(changes) MUST be len(shellCommands)
	// and their indexes match!

	Changes       Changes
	ShellCommands []string
}

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
	return chs.Filter(func(ch *Change) bool { return ch.action != ActionsDict.Noop })
}

func (chs Changes) EffectiveCount() int {
	count := 0
	for _, ch := range chs {
		if ch.action != ActionsDict.Noop {
			count++
		}
	}
	return count
}

func GetChanges(dump []bson.M, meta *ParsedMeta) (*ChangesPack, error) {
	changes := make(Changes, len(dump), len(dump))

	cmdBuilder := NewCmdBuilder("mydb", "samples")

	shellCommands := make([]string, 0)
	// todo: ensure shellComand is for EACH change

	idsMet := make(map[string]struct{})
	for i, obj := range dump {
		hashData, err := hashing.Hash(obj)
		if err != nil {
			return nil, fmt.Errorf("corrupted obj[%d] could not hash: %w", i, err)
		}
		checksumAfter := hashData.GetChecksum()

		//slog.Log(context.Background(), slog.LevelInfo, "AFTER "+hashData.String())

		id := hashData.GetIdentifier()
		idsMet[id] = struct{}{}

		if hashDataBefore, ok := meta.Lines[id]; ok {
			//slog.Log(context.Background(), slog.LevelInfo, "BEFORE "+checksumBefore)

			if hashDataBefore.GetChecksum() == checksumAfter {
				changes[i] = NewChange(hashData, ActionsDict.Noop)
			} else {
				c := NewChange(hashData, ActionsDict.Updated)
				changes[i] = c

				cmd, err := cmdBuilder.BuildShellCommand(c, obj)
				if err != nil {
					return nil, fmt.Errorf("could not build shel command: %w", err)
				}
				shellCommands = append(shellCommands, cmd)
			}
		} else {
			changes[i] = NewChange(hashData, ActionsDict.Added)
		}
	}

	for existedBeforeIdentifier, hashData := range meta.Lines {
		if _, ok := idsMet[existedBeforeIdentifier]; ok {
			continue
		}

		changes = append(changes, NewChange(hashData, ActionsDict.Deleted))
	}

	return &ChangesPack{changes, shellCommands}, nil
}
