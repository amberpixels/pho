package restore

import (
	"maps"

	"go.mongodb.org/mongo-driver/bson"
)

// cloneBsonM creates a shallow copy of bson.M to avoid mutating the original data
// This is essential for restore operations where we need to modify data without
// affecting the original document structure
func cloneBsonM(original bson.M) bson.M {
	clone := make(bson.M, len(original))
	maps.Copy(clone, original)
	return clone
}
