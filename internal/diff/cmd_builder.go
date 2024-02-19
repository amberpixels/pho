package diff

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"pho/pkg/extjson"
)

type CmdBuilder struct {
	dbName         string
	collectionName string
}

func NewCmdBuilder(dbName, collectionName string) *CmdBuilder {
	return &CmdBuilder{dbName, collectionName}
}

func (b *CmdBuilder) BuildShellCommand(c *Change, obj bson.M) (string, error) {
	switch c.action {
	case ActionsDict.Updated:
		identifiedBy, identifierValue := c.hash.GetIdentifierParts()
		marshalledValue, err := extjson.NewMarshaller(true).Marshal(obj)
		if err != nil {
			return "", fmt.Errorf("could not marshal given obj value: %w", err)
		}

		return fmt.Sprintf(`db.getCollection("%s").updateOne({%s:"%s"},{$set:%s})`,
			b.collectionName,
			identifiedBy, identifierValue,
			marshalledValue,
		), nil
	case ActionsDict.Added, ActionsDict.Deleted:
		return "", nil // not supported yet
	default:
		return "", fmt.Errorf("invalid action type")
	}

}
