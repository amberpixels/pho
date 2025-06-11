package restore

import (
	"fmt"
	"pho/internal/diff"
	"pho/pkg/extjson"
)

// MongoShellRestorer restores changes as mongo-shell commands
// It only generates command (as strings), but does not execute them
// Useful for reviewing changes, or to saving changes to be applied later manually.
type MongoShellRestorer struct {
	collectionName string
}

func NewMongoShellRestorer(collectionName string) *MongoShellRestorer {
	return &MongoShellRestorer{collectionName}
}

// Build builds a shell command for the given change
func (b *MongoShellRestorer) Build(c *diff.Change) (string, error) {
	if c == nil {
		return "", fmt.Errorf("change cannot be nil")
	}
	if c.IdentifiedBy == "" || c.IdentifierValue == "" {
		return "", fmt.Errorf("change identifiedBy+identifierValue are required fields")
	}

	switch c.Action {
	case diff.ActionsDict.Updated:

		var marshalledData []byte
		if c.Data == nil {
			return "", fmt.Errorf("updated action requires a doc")
		}

		// Clone data to avoid mutating the original
		dataClone := cloneBsonM(c.Data)
		delete(dataClone, c.IdentifiedBy)

		var err error
		if marshalledData, err = extjson.NewCanonicalMarshaller().Marshal(dataClone); err != nil {
			return "", fmt.Errorf("could not marshal given obj value: %w", err)
		}

		return fmt.Sprintf(`db.getCollection("%s").updateOne({%s:%v},{$set:%s});`,
			b.collectionName,
			c.IdentifiedBy, c.IdentifierValue,
			marshalledData,
		), nil
	case diff.ActionsDict.Added:

		var marshalledData []byte
		if c.Data != nil {
			var err error
			if marshalledData, err = extjson.NewCanonicalMarshaller().Marshal(c.Data); err != nil {
				return "", fmt.Errorf("could not marshal given obj value: %w", err)
			}
		}

		return fmt.Sprintf(`db.getCollection("%s").insertOne(%s);`,
			b.collectionName,
			marshalledData,
		), nil
	case diff.ActionsDict.Deleted:
		return fmt.Sprintf(`db.getCollection("%s").remove({"%s":%v});`,
			b.collectionName,
			c.IdentifiedBy, c.IdentifierValue,
		), nil
	case diff.ActionsDict.Noop:
		// it's considered caller not to request commands for Noop actions
		return "", ErrNoop
	default:
		return "", fmt.Errorf("invalid action type")
	}
}
