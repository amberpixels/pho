package restore

import (
	"context"
	"fmt"
	"pho/internal/diff"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

			// Clone data to avoid mutating the original
			dataClone := cloneBsonM(c.Data)
			delete(dataClone, c.IdentifiedBy)

			filter := bson.M{c.IdentifiedBy: c.IdentifierValue}
			update := bson.M{"$set": dataClone}
			result, err := b.dbCollection.UpdateOne(ctx, filter, update)
			if err != nil {
				return fmt.Errorf("mongo.UpdateOne() failed: %w", err)
			}

			// UpdateOne doesn't return ErrNoDocument as FindOne does
			// So let's return it manually, as no documents means something is wrong
			if result.MatchedCount == 0 {
				return fmt.Errorf("mongo.UpdateOne() failed: %w", mongo.ErrNoDocuments)
			}
			// TODO: keep result for future, it can provide us more things

			return nil

		case diff.ActionsDict.Added:
			if c.Data == nil {
				return fmt.Errorf("added action requires a doc")
			}

			_, err := b.dbCollection.InsertOne(ctx, c.Data)
			if err != nil {
				return fmt.Errorf("mongo.InsertOne() failed: %w", err)
			}

			return nil

		case diff.ActionsDict.Deleted:
			filter := bson.M{c.IdentifiedBy: c.IdentifierValue}
			result, err := b.dbCollection.DeleteOne(ctx, filter)
			if err != nil {
				return fmt.Errorf("mongo.DeleteOne() failed: %w", err)
			}

			// Ensure the document was actually deleted
			if result.DeletedCount == 0 {
				return fmt.Errorf("mongo.DeleteOne() failed: %w", mongo.ErrNoDocuments)
			}

			return nil

		case diff.ActionsDict.Noop:
			// No operation needed for noop actions
			return ErrNoop

		default:
			return fmt.Errorf("unknown action type: %v", c.Action)
		}
	}, nil
}
