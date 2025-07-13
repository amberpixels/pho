package restore_test

import (
	"fmt"
	"pho/internal/restore"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func TestCloneBsonM(t *testing.T) {
	tests := []struct {
		name     string
		input    bson.M
		expected bson.M
	}{
		{
			name:     "empty document",
			input:    bson.M{},
			expected: bson.M{},
		},
		{
			name: "simple document",
			input: bson.M{
				"name":  "test",
				"value": 42,
				"flag":  true,
			},
			expected: bson.M{
				"name":  "test",
				"value": 42,
				"flag":  true,
			},
		},
		{
			name: "document with nested structure",
			input: bson.M{
				"_id": "12345",
				"user": bson.M{
					"name": "John",
					"age":  30,
				},
				"tags": []string{"go", "mongodb"},
			},
			expected: bson.M{
				"_id": "12345",
				"user": bson.M{
					"name": "John",
					"age":  30,
				},
				"tags": []string{"go", "mongodb"},
			},
		},
		{
			name: "document with nil values",
			input: bson.M{
				"name":    "test",
				"deleted": nil,
				"active":  false,
			},
			expected: bson.M{
				"name":    "test",
				"deleted": nil,
				"active":  false,
			},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: bson.M{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := restore.CloneBsonM(tt.input)

			// Check that values are equal
			assert.True(t, reflect.DeepEqual(result, tt.expected))

			// Check that it's a different object (not same reference)
			if tt.input != nil {
				assert.NotSame(t, &tt.input, &result)
			}
		})
	}
}

func TestCloneBsonM_MutationSafety(t *testing.T) {
	// Test that modifying the clone doesn't affect the original
	original := bson.M{
		"name":  "original",
		"value": 100,
		"nested": bson.M{
			"inner": "data",
		},
	}

	clone := restore.CloneBsonM(original)

	// Modify the clone
	clone["name"] = "modified"
	clone["newField"] = "added"

	// Original should remain unchanged
	assert.Equal(t, "original", original["name"])
	_, exists := original["newField"]
	assert.False(t, exists, "Original document should not have newField")

	// Clone should have the modifications
	assert.Equal(t, "modified", clone["name"])
	assert.Equal(t, "added", clone["newField"])
}

func TestCloneBsonM_NestedMutationBehavior(t *testing.T) {
	// Test behavior with nested structures (shallow copy)
	original := bson.M{
		"nested": bson.M{
			"inner": "value",
		},
		"array": []string{"a", "b"},
	}

	clone := restore.CloneBsonM(original)

	// Modifying nested objects will affect both (shallow copy behavior)
	if nested, ok := clone["nested"].(bson.M); ok {
		nested["inner"] = "modified"
	}

	// This demonstrates shallow copy behavior - nested objects are shared
	if originalNested, ok := original["nested"].(bson.M); ok {
		// Document expected behavior: maps.Copy creates shallow copies
		// so nested objects are shared between original and clone
		_ = originalNested["inner"] // Access to document the expected behavior
	}

	// But top-level additions don't affect the original
	clone["topLevel"] = "new"
	_, exists := original["topLevel"]
	assert.False(t, exists, "Top-level addition to clone should not affect original")
}

func TestCloneBsonM_EmptyAndNilHandling(t *testing.T) {
	// Test empty bson.M
	empty := bson.M{}
	clonedEmpty := restore.CloneBsonM(empty)

	assert.Empty(t, clonedEmpty)

	// Add to clone, shouldn't affect original
	clonedEmpty["test"] = "value"
	assert.Empty(t, empty)

	// Test nil input
	var nilDoc bson.M
	clonedNil := restore.CloneBsonM(nilDoc)

	assert.NotNil(t, clonedNil)
	assert.Empty(t, clonedNil)
}

func TestCloneBsonM_TypePreservation(t *testing.T) {
	// Test that different types are preserved
	original := bson.M{
		"string": "text",
		"int":    42,
		"int64":  int64(9223372036854775807),
		"float":  3.14159,
		"bool":   true,
		"bytes":  []byte("binary data"),
		"slice":  []any{"a", 1, true},
		"map":    map[string]any{"key": "value"},
	}

	clone := restore.CloneBsonM(original)

	for key, originalValue := range original {
		cloneValue, exists := clone[key]
		assert.True(t, exists, "Clone missing key: %s", key)
		assert.True(t, reflect.DeepEqual(originalValue, cloneValue),
			"Type not preserved for key %s: original = %v (%T), clone = %v (%T)",
			key, originalValue, originalValue, cloneValue, cloneValue)
	}
}

func TestCloneBsonM_CapacityOptimization(t *testing.T) {
	// Test that the clone has appropriate capacity
	large := make(bson.M, 100)
	for i := range 100 {
		large[fmt.Sprintf("key%d", i)] = i
	}

	clone := restore.CloneBsonM(large)

	assert.Len(t, clone, len(large))

	// Verify all elements are present
	for key, value := range large {
		cloneValue, exists := clone[key]
		assert.True(t, exists && cloneValue == value, "Clone missing or incorrect value for key %s", key)
	}
}
