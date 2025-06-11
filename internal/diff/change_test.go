package diff

import (
	"pho/internal/hashing"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewChange(t *testing.T) {
	tests := []struct {
		name            string
		identifiedBy    string
		identifierValue interface{}
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
			
			if change == nil {
				t.Fatal("NewChange() returned nil")
			}
			
			if change.IdentifiedBy != tt.identifiedBy {
				t.Errorf("IdentifiedBy = %v, want %v", change.IdentifiedBy, tt.identifiedBy)
			}
			
			if change.IdentifierValue != tt.identifierValue {
				t.Errorf("IdentifierValue = %v, want %v", change.IdentifierValue, tt.identifierValue)
			}
			
			if change.Action != tt.action {
				t.Errorf("Action = %v, want %v", change.Action, tt.action)
			}
			
			if tt.expectData {
				if change.Data == nil {
					t.Error("Expected data, got nil")
				} else if len(tt.data) > 0 {
					// Check first element
					for key, value := range tt.data[0] {
						if change.Data[key] != value {
							t.Errorf("Data[%s] = %v, want %v", key, change.Data[key], value)
						}
					}
				}
			} else {
				if change.Data != nil {
					t.Error("Expected no data, got data")
				}
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
			
			if result != tt.expected {
				t.Errorf("IsEffective() = %v, want %v", result, tt.expected)
			}
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
	if len := changes.Len(); len != expected {
		t.Errorf("Len() = %v, want %v", len, expected)
	}
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
	
	if len(filtered) != 2 {
		t.Errorf("Filter() returned %d changes, want 2", len(filtered))
	}
	
	for _, change := range filtered {
		if change.Action != ActionsDict.Added && change.Action != ActionsDict.Updated {
			t.Errorf("Filter() returned unexpected action: %v", change.Action)
		}
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
	if len(addedChanges) != 2 {
		t.Errorf("FilterByAction(Added) returned %d changes, want 2", len(addedChanges))
	}
	
	updatedChanges := changes.FilterByAction(ActionsDict.Updated)
	if len(updatedChanges) != 1 {
		t.Errorf("FilterByAction(Updated) returned %d changes, want 1", len(updatedChanges))
	}
	
	deletedChanges := changes.FilterByAction(ActionsDict.Deleted)
	if len(deletedChanges) != 1 {
		t.Errorf("FilterByAction(Deleted) returned %d changes, want 1", len(deletedChanges))
	}
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
	
	if len(effective) != 3 {
		t.Errorf("EffectiveOnes() returned %d changes, want 3", len(effective))
	}
	
	for _, change := range effective {
		if change.Action == ActionsDict.Noop {
			t.Error("EffectiveOnes() should not include Noop actions")
		}
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
	if err != nil {
		t.Fatalf("CalculateChanges() error = %v", err)
	}
	
	if len(changes) != 5 { // 4 destination docs + 1 deleted
		t.Errorf("CalculateChanges() returned %d changes, want 5", len(changes))
	}
	
	// Analyze changes
	effective := changes.EffectiveOnes()
	
	// Should have: 1 added (doc4), 1 updated (doc3), 1 deleted (doc5)
	expectedEffective := 3
	if len(effective) != expectedEffective {
		t.Errorf("Expected %d effective changes, got %d", expectedEffective, len(effective))
	}
	
	// Count by action
	added := changes.FilterByAction(ActionsDict.Added)
	updated := changes.FilterByAction(ActionsDict.Updated)
	deleted := changes.FilterByAction(ActionsDict.Deleted)
	noop := changes.FilterByAction(ActionsDict.Noop)
	
	if len(added) != 1 {
		t.Errorf("Expected 1 added change, got %d", len(added))
	}
	
	if len(updated) != 1 {
		t.Errorf("Expected 1 updated change, got %d", len(updated))
	}
	
	if len(deleted) != 1 {
		t.Errorf("Expected 1 deleted change, got %d", len(deleted))
	}
	
	if len(noop) != 2 {
		t.Errorf("Expected 2 noop changes, got %d", len(noop))
	}
}

func TestCalculateChanges_EmptySource(t *testing.T) {
	// All documents are new
	source := make(map[string]*hashing.HashData)
	destination := []bson.M{
		{"_id": "new1", "name": "New Document 1"},
		{"_id": "new2", "name": "New Document 2"},
	}
	
	changes, err := CalculateChanges(source, destination)
	if err != nil {
		t.Fatalf("CalculateChanges() error = %v", err)
	}
	
	if len(changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(changes))
	}
	
	for _, change := range changes {
		if change.Action != ActionsDict.Added {
			t.Errorf("All changes should be Added, got %v", change.Action)
		}
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
	if err != nil {
		t.Fatalf("CalculateChanges() error = %v", err)
	}
	
	if len(changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(changes))
	}
	
	for _, change := range changes {
		if change.Action != ActionsDict.Deleted {
			t.Errorf("All changes should be Deleted, got %v", change.Action)
		}
	}
}

func TestCalculateChanges_InvalidDocument(t *testing.T) {
	// Document without _id should cause error
	source := make(map[string]*hashing.HashData)
	destination := []bson.M{
		{"name": "Document without ID"},
	}
	
	_, err := CalculateChanges(source, destination)
	if err == nil {
		t.Error("CalculateChanges() should return error for document without _id")
	}
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
	if err != nil {
		t.Fatalf("CalculateChanges() error = %v", err)
	}
	
	if len(changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(changes))
	}
	
	// One should be noop, one should be added
	effective := changes.EffectiveOnes()
	if len(effective) != 1 {
		t.Errorf("Expected 1 effective change, got %d", len(effective))
	}
	
	if effective[0].Action != ActionsDict.Added {
		t.Errorf("Expected Added action, got %v", effective[0].Action)
	}
}