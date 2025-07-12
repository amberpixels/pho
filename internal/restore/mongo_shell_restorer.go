package restore

import (
	"errors"
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

// Build builds a shell command for the given change.
func (b *MongoShellRestorer) Build(c *diff.Change) (string, error) {
	if c == nil {
		return "", errors.New("change cannot be nil")
	}
	if c.IdentifiedBy == "" || c.IdentifierValue == "" {
		return "", errors.New("change identifiedBy+identifierValue are required fields")
	}

	switch c.Action {
	case diff.ActionUpdated:

		var marshalledData []byte
		if c.Data == nil {
			return "", errors.New("updated action requires a doc")
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
	case diff.ActionAdded:

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
	case diff.ActionDeleted:
		return fmt.Sprintf(`db.getCollection("%s").remove({"%s":%v});`,
			b.collectionName,
			c.IdentifiedBy, c.IdentifierValue,
		), nil
	case diff.ActionNoop:
		// it's considered caller not to request commands for Noop actions
		return "", ErrNoop
	default:
		return "", errors.New("invalid action type")
	}
}
