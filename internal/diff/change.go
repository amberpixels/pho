package diff

import (
	"fmt"
	"pho/internal/hashing"

	"go.mongodb.org/mongo-driver/bson"
)

// Change holds information about one document change
// It stores data enough to perform the change
// Note: we do not need original document state (as we perform only FullUpdates only).
type Change struct {
	// Action that was applied
	Action Action

	// Data is the data changed (for Action=Updated/Added)
	Data bson.M

	IdentifiedBy    string
	IdentifierValue any
}

func NewChange(identifiedBy string, identifierValue any, action Action, data ...bson.M) *Change {
	change := &Change{IdentifiedBy: identifiedBy, IdentifierValue: identifierValue, Action: action}
	if len(data) > 0 {
		change.Data = data[0]
	}

	return change
}

type Changes []*Change

func (ch *Change) IsEffective() bool {
	return ch.Action.IsEffective()
}

func (chs Changes) Len() int { return len(chs) }

// Filter returns a filtered list of changes (by a given filter func).
func (chs Changes) Filter(f func(*Change) bool) Changes {
	var filtered Changes
	for _, ch := range chs {
		if f(ch) {
			filtered = append(filtered, ch)
		}
	}

	return filtered
}

// FilterByAction returns a filtered list of changes by action type.
func (chs Changes) FilterByAction(a Action) Changes {
	return chs.Filter(func(change *Change) bool {
		return change.Action == a
	})
}

// EffectiveOnes is an alias for Filter(IsEffective).
func (chs Changes) EffectiveOnes() Changes {
	return chs.Filter(func(ch *Change) bool { return ch.IsEffective() })
}

// CalculateChanges calculates changes that represent difference between
// given `source` hashed lines and `destination` list of current versions of documents.
func CalculateChanges(source map[string]*hashing.HashData, destination []bson.M) (Changes, error) {
	n := len(destination)
	changes := make(Changes, 0, n+len(source)) // Pre-allocate with capacity for worst case

	// hashmap for documents that were processed
	idsLUT := make(map[string]struct{})
	for i, doc := range destination {
		hashData, err := hashing.Hash(doc)
		if err != nil {
			return nil, fmt.Errorf("corrupted obj[%d] could not hash: %w", i, err)
		}

		// Using full _id::1 identifier as a LUT key
		id := hashData.GetIdentifier()
		idsLUT[id] = struct{}{}

		identifiedBy, identifierValue := hashData.GetIdentifierParts()
		checksumAfter := hashData.GetChecksum()

		// Check if not found in source, so it's a new document
		hashDataBefore, ok := source[id]
		if !ok {
			changes = append(changes, NewChange(identifiedBy, identifierValue, ActionAdded, doc))
			continue
		}

		// Document was not change, so it's a nothing
		if hashDataBefore.GetChecksum() == checksumAfter {
			changes = append(changes, NewChange(identifiedBy, identifierValue, ActionNoop))
			continue
		}

		// Otherwise it was an update:
		changes = append(changes, NewChange(identifiedBy, identifierValue, ActionUpdated, doc))
	}

	// To get delete changes we have to do the other way round:
	// Source => Destination
	// Note: order of deletion documents will not be respected
	for sourceDocIdentifier, hashData := range source {
		if _, ok := idsLUT[sourceDocIdentifier]; ok {
			continue
		}

		// Skip if hashData is nil or malformed
		if hashData == nil {
			continue
		}

		// we can either parse sourceDocIdentifier
		// or take it again from hashData
		identifiedBy, identifierValue := hashData.GetIdentifierParts()

		changes = append(changes, NewChange(identifiedBy, identifierValue, ActionDeleted))
	}

	return changes, nil
}
