package restore

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"pho/internal/diff"
)

// MongoClientRestorer restores changes via mongo go client
type MongoClientRestorer struct {
	dbCollection *mongo.Collection
}

func NewMongoClientRestorer(dbCollection *mongo.Collection) *MongoClientRestorer {
	return &MongoClientRestorer{dbCollection}
}

func (b *MongoClientRestorer) Build(c *diff.Change) (func(ctx context.Context) error, error) {
	if c.IdentifiedBy == "" || c.IdentifierValue == "" {
		return nil, fmt.Errorf("change identifiedBy+identifierValue are required fields")
	}
	if b.dbCollection == nil {
		return nil, fmt.Errorf("connected db collection is required")
	}

	return func(ctx context.Context) error {
		switch c.Action {
		case diff.ActionsDict.Updated:
			if c.Data == nil {
				return fmt.Errorf("updated action requires a doc")
			}

			// TODO c.Data needs to be cloned here, so it's not mutated
			delete(c.Data, c.IdentifiedBy)

			resultForFuture, err := b.dbCollection.UpdateOne(ctx,
				bson.M{c.IdentifiedBy: c.IdentifierValue},
				bson.M{"$set": c.Data},
			)

			// TODO: do no loose UpdateResult, we can use it to give more verbosity to users
			_ = resultForFuture

			return err
		default:
			// todo: implement other cases
			return fmt.Errorf("not implemented")
		}
	}, nil
}
