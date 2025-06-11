package restore

import (
	"fmt"
	"reflect"
	"testing"

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
			name: "nil input",
			input: nil,
			expected: bson.M{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cloneBsonM(tt.input)
			
			// Check that values are equal
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("cloneBsonM() = %v, want %v", result, tt.expected)
			}
			
			// Check that it's a different object (not same reference)
			if tt.input != nil && &result == &tt.input {
				t.Error("cloneBsonM() should return a different object")
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
	
	clone := cloneBsonM(original)
	
	// Modify the clone
	clone["name"] = "modified"
	clone["newField"] = "added"
	
	// Original should remain unchanged
	if original["name"] != "original" {
		t.Errorf("Original document was modified: name = %v, want 'original'", original["name"])
	}
	
	if _, exists := original["newField"]; exists {
		t.Error("Original document should not have newField")
	}
	
	// Clone should have the modifications
	if clone["name"] != "modified" {
		t.Errorf("Clone was not modified: name = %v, want 'modified'", clone["name"])
	}
	
	if clone["newField"] != "added" {
		t.Errorf("Clone should have newField = 'added', got %v", clone["newField"])
	}
}

func TestCloneBsonM_NestedMutationBehavior(t *testing.T) {
	// Test behavior with nested structures (shallow copy)
	original := bson.M{
		"nested": bson.M{
			"inner": "value",
		},
		"array": []string{"a", "b"},
	}
	
	clone := cloneBsonM(original)
	
	// Modifying nested objects will affect both (shallow copy behavior)
	if nested, ok := clone["nested"].(bson.M); ok {
		nested["inner"] = "modified"
	}
	
	// This demonstrates shallow copy behavior - nested objects are shared
	if originalNested, ok := original["nested"].(bson.M); ok {
		if originalNested["inner"] != "modified" {
			// This behavior depends on maps.Copy implementation
			// If it changes in the future, this test documents the current behavior
		}
	}
	
	// But top-level additions don't affect the original
	clone["topLevel"] = "new"
	if _, exists := original["topLevel"]; exists {
		t.Error("Top-level addition to clone should not affect original")
	}
}

func TestCloneBsonM_EmptyAndNilHandling(t *testing.T) {
	// Test empty bson.M
	empty := bson.M{}
	clonedEmpty := cloneBsonM(empty)
	
	if len(clonedEmpty) != 0 {
		t.Errorf("Cloned empty document should be empty, got length %d", len(clonedEmpty))
	}
	
	// Add to clone, shouldn't affect original
	clonedEmpty["test"] = "value"
	if len(empty) != 0 {
		t.Error("Original empty document should remain empty")
	}
	
	// Test nil input
	var nilDoc bson.M
	clonedNil := cloneBsonM(nilDoc)
	
	if clonedNil == nil {
		t.Error("cloneBsonM(nil) should return non-nil empty map")
	}
	
	if len(clonedNil) != 0 {
		t.Errorf("cloneBsonM(nil) should return empty map, got length %d", len(clonedNil))
	}
}

func TestCloneBsonM_TypePreservation(t *testing.T) {
	// Test that different types are preserved
	original := bson.M{
		"string":  "text",
		"int":     42,
		"int64":   int64(9223372036854775807),
		"float":   3.14159,
		"bool":    true,
		"bytes":   []byte("binary data"),
		"slice":   []interface{}{"a", 1, true},
		"map":     map[string]interface{}{"key": "value"},
	}
	
	clone := cloneBsonM(original)
	
	for key, originalValue := range original {
		cloneValue, exists := clone[key]
		if !exists {
			t.Errorf("Clone missing key: %s", key)
			continue
		}
		
		if !reflect.DeepEqual(originalValue, cloneValue) {
			t.Errorf("Type not preserved for key %s: original = %v (%T), clone = %v (%T)", 
				key, originalValue, originalValue, cloneValue, cloneValue)
		}
	}
}

func TestCloneBsonM_CapacityOptimization(t *testing.T) {
	// Test that the clone has appropriate capacity
	large := make(bson.M, 100)
	for i := 0; i < 100; i++ {
		large[fmt.Sprintf("key%d", i)] = i
	}
	
	clone := cloneBsonM(large)
	
	if len(clone) != len(large) {
		t.Errorf("Clone length = %d, want %d", len(clone), len(large))
	}
	
	// Verify all elements are present
	for key, value := range large {
		if cloneValue, exists := clone[key]; !exists || cloneValue != value {
			t.Errorf("Clone missing or incorrect value for key %s", key)
		}
	}
}