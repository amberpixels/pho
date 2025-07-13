package restore

import (
	"context"
	"errors"
	"fmt"
	"pho/internal/diff"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoClientRestorer restores changes via mongo go client.
type MongoClientRestorer struct {
	dbCollection *mongo.Collection
}

func NewMongoClientRestorer(dbCollection *mongo.Collection) *MongoClientRestorer {
	return &MongoClientRestorer{dbCollection}
}

func (r *MongoClientRestorer) GetDBCollection() *mongo.Collection { return r.dbCollection }

func (r *MongoClientRestorer) Build(c *diff.Change) (func(ctx context.Context) error, error) {
	if c == nil {
		return nil, errors.New("change cannot be nil")
	}
	if c.IdentifiedBy == "" || c.IdentifierValue == "" {
		return nil, errors.New("change identifiedBy+identifierValue are required fields")
	}
	if r.dbCollection == nil {
		return nil, errors.New("connected db collection is required")
	}

	return func(ctx context.Context) error {
		switch c.Action {
		case diff.ActionUpdated:
			if c.Data == nil {
				return errors.New("updated action requires a doc")
			}

			// Clone data to avoid mutating the original
			dataClone := cloneBsonM(c.Data)
			delete(dataClone, c.IdentifiedBy)

			filter := bson.M{c.IdentifiedBy: c.IdentifierValue}
			update := bson.M{"$set": dataClone}
			result, err := r.dbCollection.UpdateOne(ctx, filter, update)
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

		case diff.ActionAdded:
			if c.Data == nil {
				return errors.New("added action requires a doc")
			}

			_, err := r.dbCollection.InsertOne(ctx, c.Data)
			if err != nil {
				return fmt.Errorf("mongo.InsertOne() failed: %w", err)
			}

			return nil

		case diff.ActionDeleted:
			filter := bson.M{c.IdentifiedBy: c.IdentifierValue}
			result, err := r.dbCollection.DeleteOne(ctx, filter)
			if err != nil {
				return fmt.Errorf("mongo.DeleteOne() failed: %w", err)
			}

			// Ensure the document was actually deleted
			if result.DeletedCount == 0 {
				return fmt.Errorf("mongo.DeleteOne() failed: %w", mongo.ErrNoDocuments)
			}

			return nil

		case diff.ActionNoop:
			// No operation needed for noop actions
			return ErrNoop

		default:
			return fmt.Errorf("unknown action type: %v", c.Action)
		}
	}, nil
}
