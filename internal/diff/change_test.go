package diff

import (
	"pho/internal/hashing"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewChange(t *testing.T) {
	tests := []struct {
		name            string
		identifiedBy    string
		identifierValue any
		action          Action
		data            []bson.M
		expectData      bool
	}{
		{
			name:            "change without data",
			identifiedBy:    "_id",
			identifierValue: "test-1",
			action:          ActionsDict.Deleted,
			data:            nil,
			expectData:      false,
		},
		{
			name:            "change with data",
			identifiedBy:    "_id",
			identifierValue: "test-2",
			action:          ActionsDict.Updated,
			data:            []bson.M{{"name": "updated"}},
			expectData:      true,
		},
		{
			name:            "change with multiple data (only first used)",
			identifiedBy:    "_id",
			identifierValue: "test-3",
			action:          ActionsDict.Added,
			data:            []bson.M{{"name": "first"}, {"name": "second"}},
			expectData:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := NewChange(tt.identifiedBy, tt.identifierValue, tt.action, tt.data...)

			assert.NotNil(t, change)
			assert.Equal(t, tt.identifiedBy, change.IdentifiedBy)
			assert.Equal(t, tt.identifierValue, change.IdentifierValue)
			assert.Equal(t, tt.action, change.Action)

			if tt.expectData {
				assert.NotNil(t, change.Data)
				if len(tt.data) > 0 {
					// Check first element
					for key, value := range tt.data[0] {
						assert.Equal(t, value, change.Data[key])
					}
				}
			} else {
				assert.Nil(t, change.Data)
			}
		})
	}
}

func TestChange_IsEffective(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected bool
	}{
		{
			name:     "noop action is not effective",
			action:   ActionsDict.Noop,
			expected: false,
		},
		{
			name:     "added action is effective",
			action:   ActionsDict.Added,
			expected: true,
		},
		{
			name:     "updated action is effective",
			action:   ActionsDict.Updated,
			expected: true,
		},
		{
			name:     "deleted action is effective",
			action:   ActionsDict.Deleted,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := &Change{Action: tt.action}
			result := change.IsEffective()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChanges_Len(t *testing.T) {
	changes := Changes{
		NewChange("_id", "1", ActionsDict.Added),
		NewChange("_id", "2", ActionsDict.Updated),
		NewChange("_id", "3", ActionsDict.Deleted),
	}

	expected := 3
	assert.Equal(t, expected, changes.Len())
}

func TestChanges_Filter(t *testing.T) {
	changes := Changes{
		NewChange("_id", "1", ActionsDict.Added),
		NewChange("_id", "2", ActionsDict.Updated),
		NewChange("_id", "3", ActionsDict.Deleted),
		NewChange("_id", "4", ActionsDict.Noop),
	}

	// Filter for only Added and Updated
	filtered := changes.Filter(func(c *Change) bool {
		return c.Action == ActionsDict.Added || c.Action == ActionsDict.Updated
	})

	assert.Len(t, filtered, 2)

	for _, change := range filtered {
		assert.True(t, change.Action == ActionsDict.Added || change.Action == ActionsDict.Updated)
	}
}

func TestChanges_FilterByAction(t *testing.T) {
	changes := Changes{
		NewChange("_id", "1", ActionsDict.Added),
		NewChange("_id", "2", ActionsDict.Updated),
		NewChange("_id", "3", ActionsDict.Added),
		NewChange("_id", "4", ActionsDict.Deleted),
	}

	addedChanges := changes.FilterByAction(ActionsDict.Added)
	assert.Len(t, addedChanges, 2)

	updatedChanges := changes.FilterByAction(ActionsDict.Updated)
	assert.Len(t, updatedChanges, 1)

	deletedChanges := changes.FilterByAction(ActionsDict.Deleted)
	assert.Len(t, deletedChanges, 1)
}

func TestChanges_EffectiveOnes(t *testing.T) {
	changes := Changes{
		NewChange("_id", "1", ActionsDict.Added),
		NewChange("_id", "2", ActionsDict.Noop),
		NewChange("_id", "3", ActionsDict.Updated),
		NewChange("_id", "4", ActionsDict.Noop),
		NewChange("_id", "5", ActionsDict.Deleted),
	}

	effective := changes.EffectiveOnes()

	assert.Len(t, effective, 3)

	for _, change := range effective {
		assert.NotEqual(t, ActionsDict.Noop, change.Action)
	}
}

func TestCalculateChanges(t *testing.T) {
	// Create test documents
	doc1 := bson.M{"_id": "doc1", "name": "Document 1", "value": 100}
	doc2 := bson.M{"_id": "doc2", "name": "Document 2", "value": 200}
	doc3 := bson.M{"_id": "doc3", "name": "Document 3 Modified", "value": 300}
	doc4 := bson.M{"_id": "doc4", "name": "Document 4 New", "value": 400}

	// Create source hashes (simulating original state)
	source := make(map[string]*hashing.HashData)

	// doc1 and doc2 unchanged, doc3 will be modified, doc5 will be deleted
	hash1, _ := hashing.Hash(doc1)
	hash2, _ := hashing.Hash(doc2)
	originalDoc3 := bson.M{"_id": "doc3", "name": "Document 3", "value": 300}
	hash3, _ := hashing.Hash(originalDoc3)
	deletedDoc := bson.M{"_id": "doc5", "name": "Document 5 Deleted", "value": 500}
	hash5, _ := hashing.Hash(deletedDoc)

	source[hash1.GetIdentifier()] = hash1
	source[hash2.GetIdentifier()] = hash2
	source[hash3.GetIdentifier()] = hash3
	source[hash5.GetIdentifier()] = hash5

	// Current destination (after editing)
	destination := []bson.M{doc1, doc2, doc3, doc4}

	changes, err := CalculateChanges(source, destination)
	require.NoError(t, err)
	assert.Len(t, changes, 5) // 4 destination docs + 1 deleted

	// Analyze changes
	effective := changes.EffectiveOnes()

	// Should have: 1 added (doc4), 1 updated (doc3), 1 deleted (doc5)
	expectedEffective := 3
	assert.Len(t, effective, expectedEffective)

	// Count by action
	added := changes.FilterByAction(ActionsDict.Added)
	updated := changes.FilterByAction(ActionsDict.Updated)
	deleted := changes.FilterByAction(ActionsDict.Deleted)
	noop := changes.FilterByAction(ActionsDict.Noop)

	assert.Len(t, added, 1)
	assert.Len(t, updated, 1)
	assert.Len(t, deleted, 1)
	assert.Len(t, noop, 2)
}

func TestCalculateChanges_EmptySource(t *testing.T) {
	// All documents are new
	source := make(map[string]*hashing.HashData)
	destination := []bson.M{
		{"_id": "new1", "name": "New Document 1"},
		{"_id": "new2", "name": "New Document 2"},
	}

	changes, err := CalculateChanges(source, destination)
	require.NoError(t, err)
	assert.Len(t, changes, 2)

	for _, change := range changes {
		assert.Equal(t, ActionsDict.Added, change.Action)
	}
}

func TestCalculateChanges_EmptyDestination(t *testing.T) {
	// All documents are deleted
	doc1 := bson.M{"_id": "deleted1", "name": "Deleted Document 1"}
	doc2 := bson.M{"_id": "deleted2", "name": "Deleted Document 2"}

	source := make(map[string]*hashing.HashData)
	hash1, _ := hashing.Hash(doc1)
	hash2, _ := hashing.Hash(doc2)
	source[hash1.GetIdentifier()] = hash1
	source[hash2.GetIdentifier()] = hash2

	destination := []bson.M{}

	changes, err := CalculateChanges(source, destination)
	require.NoError(t, err)
	assert.Len(t, changes, 2)

	for _, change := range changes {
		assert.Equal(t, ActionsDict.Deleted, change.Action)
	}
}

func TestCalculateChanges_InvalidDocument(t *testing.T) {
	// Document without _id should cause error
	source := make(map[string]*hashing.HashData)
	destination := []bson.M{
		{"name": "Document without ID"},
	}

	_, err := CalculateChanges(source, destination)
	assert.Error(t, err)
}

func TestCalculateChanges_ObjectIDSupport(t *testing.T) {
	// Test with ObjectID identifiers
	oid1 := primitive.NewObjectID()
	oid2 := primitive.NewObjectID()

	doc1 := bson.M{"_id": oid1, "name": "Document with ObjectID 1"}
	doc2 := bson.M{"_id": oid2, "name": "Document with ObjectID 2"}

	source := make(map[string]*hashing.HashData)
	hash1, _ := hashing.Hash(doc1)
	source[hash1.GetIdentifier()] = hash1

	// doc2 is new
	destination := []bson.M{doc1, doc2}

	changes, err := CalculateChanges(source, destination)
	require.NoError(t, err)
	assert.Len(t, changes, 2)

	// One should be noop, one should be added
	effective := changes.EffectiveOnes()
	assert.Len(t, effective, 1)
	assert.Equal(t, ActionsDict.Added, effective[0].Action)
}
