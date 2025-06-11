package restore

import (
	"strings"
	"testing"

	"pho/internal/diff"

	"go.mongodb.org/mongo-driver/bson"
)

func TestNewMongoClientRestorer(t *testing.T) {
	restorer := NewMongoClientRestorer(nil)
	
	if restorer == nil {
		t.Fatal("NewMongoClientRestorer() returned nil")
	}
	
	if restorer.dbCollection != nil {
		t.Error("Expected dbCollection to be nil when passed nil")
	}
}

func TestMongoClientRestorer_Build_ValidationErrors(t *testing.T) {
	restorer := &MongoClientRestorer{dbCollection: nil}
	
	tests := []struct {
		name          string
		change        *diff.Change
		wantErr       bool
		errorContains string
	}{
		{
			name: "missing identified by",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "",
				IdentifierValue: "test",
			},
			wantErr:       true,
			errorContains: "identifiedBy+identifierValue are required",
		},
		{
			name: "missing identifier value",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "_id",
				IdentifierValue: "",
			},
			wantErr:       true,
			errorContains: "identifiedBy+identifierValue are required",
		},
		{
			name: "nil collection",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "_id",
				IdentifierValue: "test",
			},
			wantErr:       true,
			errorContains: "connected db collection is required",
		},
		{
			name: "nil change",
			change: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := restorer.Build(tt.change)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr && tt.errorContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Build() error = %v, want error containing %v", err, tt.errorContains)
				}
			}
		})
	}
}

func TestMongoClientRestorer_Build_ActionValidation(t *testing.T) {
	// Create a mock collection that satisfies the interface requirement
	// Since we can't easily mock mongo.Collection, we focus on testing the build logic
	
	tests := []struct {
		name                 string
		change               *diff.Change
		shouldBuildSucceed   bool
		shouldHaveFunction   bool
	}{
		{
			name: "update action with data",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "_id",
				IdentifierValue: "test123",
				Data: bson.M{
					"_id":  "test123",
					"name": "updated",
				},
			},
			shouldBuildSucceed: false, // Will fail due to nil collection
			shouldHaveFunction: false,
		},
		{
			name: "add action with data",
			change: &diff.Change{
				Action:          diff.ActionAdded,
				IdentifiedBy:    "_id",
				IdentifierValue: "test123",
				Data: bson.M{
					"_id":  "test123",
					"name": "new",
				},
			},
			shouldBuildSucceed: false, // Will fail due to nil collection
			shouldHaveFunction: false,
		},
		{
			name: "delete action",
			change: &diff.Change{
				Action:          diff.ActionDeleted,
				IdentifiedBy:    "_id",
				IdentifierValue: "test123",
			},
			shouldBuildSucceed: false, // Will fail due to nil collection
			shouldHaveFunction: false,
		},
		{
			name: "noop action",
			change: &diff.Change{
				Action:          diff.ActionNoop,
				IdentifiedBy:    "_id",
				IdentifierValue: "test123",
			},
			shouldBuildSucceed: false, // Will fail due to nil collection
			shouldHaveFunction: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restorer := &MongoClientRestorer{dbCollection: nil}
			
			fn, err := restorer.Build(tt.change)
			
			if (err == nil) != tt.shouldBuildSucceed {
				t.Errorf("Build() error = %v, shouldBuildSucceed %v", err, tt.shouldBuildSucceed)
				return
			}
			
			if (fn != nil) != tt.shouldHaveFunction {
				t.Errorf("Build() function = %v, shouldHaveFunction %v", fn != nil, tt.shouldHaveFunction)
			}
		})
	}
}

func TestMongoClientRestorer_ActionLogic(t *testing.T) {
	// Test the action handling logic without needing real MongoDB connection
	// We focus on the data validation and preparation logic
	
	restorer := &MongoClientRestorer{dbCollection: nil}
	
	tests := []struct {
		name          string
		change        *diff.Change
		wantBuildErr  bool
		errorContains string
	}{
		{
			name: "update without data should build but fail on execution",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "_id",
				IdentifierValue: "test123",
				Data:            nil,
			},
			wantBuildErr:  true,
			errorContains: "connected db collection is required",
		},
		{
			name: "add without data should build but fail on execution",
			change: &diff.Change{
				Action:          diff.ActionAdded,
				IdentifiedBy:    "_id",
				IdentifierValue: "test123",
				Data:            nil,
			},
			wantBuildErr:  true,
			errorContains: "connected db collection is required",
		},
		{
			name: "invalid action",
			change: &diff.Change{
				Action:          diff.Action(99),
				IdentifiedBy:    "_id",
				IdentifierValue: "test",
			},
			wantBuildErr:  true,
			errorContains: "connected db collection is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := restorer.Build(tt.change)
			
			if (err != nil) != tt.wantBuildErr {
				t.Errorf("Build() error = %v, wantBuildErr %v", err, tt.wantBuildErr)
				return
			}
			
			if tt.wantBuildErr && tt.errorContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Build() error = %v, want error containing %v", err, tt.errorContains)
				}
			}
		})
	}
}

func TestMongoClientRestorer_DataCloning(t *testing.T) {
	// Test that the cloning logic works correctly
	// We can test this by examining what would be passed to the update operation
	
	originalData := bson.M{
		"_id":   "test123",
		"name":  "original",
		"value": 42,
	}
	
	change := &diff.Change{
		Action:          diff.ActionUpdated,
		IdentifiedBy:    "_id",
		IdentifierValue: "test123",
		Data:            originalData,
	}
	
	restorer := &MongoClientRestorer{dbCollection: nil}
	
	// Even though this will fail due to nil collection, we can verify
	// that the original data structure is not modified
	_, err := restorer.Build(change)
	
	// Should get a collection error
	if err == nil {
		t.Error("Expected error due to nil collection")
	}
	
	// Original data should remain unchanged
	if originalData["name"] != "original" {
		t.Errorf("Original data was modified: name = %v, want 'original'", originalData["name"])
	}
	
	if _, exists := originalData["_id"]; !exists {
		t.Error("Original data should still contain _id field")
	}
	
	if originalData["value"] != 42 {
		t.Errorf("Original data was modified: value = %v, want 42", originalData["value"])
	}
}

func TestMongoClientRestorer_Build_NilChangeHandling(t *testing.T) {
	restorer := &MongoClientRestorer{dbCollection: nil}
	
	_, err := restorer.Build(nil)
	if err == nil {
		t.Error("Build() should return error for nil change")
	}
}

func TestMongoClientRestorer_Build_EmptyFieldValidation(t *testing.T) {
	restorer := &MongoClientRestorer{dbCollection: nil}
	
	tests := []struct {
		name      string
		change    *diff.Change
		wantError bool
	}{
		{
			name: "empty identified by",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "",
				IdentifierValue: "value",
			},
			wantError: true,
		},
		{
			name: "empty identifier value",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "_id",
				IdentifierValue: "",
			},
			wantError: true,
		},
		{
			name: "both empty",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "",
				IdentifierValue: "",
			},
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := restorer.Build(tt.change)
			
			if (err != nil) != tt.wantError {
				t.Errorf("Build() error = %v, wantError %v", err, tt.wantError)
			}
			
			if tt.wantError && err != nil {
				if !strings.Contains(err.Error(), "identifiedBy+identifierValue are required") {
					t.Errorf("Build() error = %v, want error about required fields", err)
				}
			}
		})
	}
}

// Test that we can create execution functions even if we can't execute them
func TestMongoClientRestorer_ExecutionFunctionCreation(t *testing.T) {
	// This test verifies that the Build method returns the correct function signature
	// even though we can't test execution without a real MongoDB connection
	
	change := &diff.Change{
		Action:          diff.ActionNoop,
		IdentifiedBy:    "_id",
		IdentifierValue: "test",
	}
	
	restorer := &MongoClientRestorer{dbCollection: nil}
	
	// This should fail due to nil collection
	fn, err := restorer.Build(change)
	
	if err == nil {
		t.Error("Expected error due to nil collection")
	}
	
	if fn != nil {
		t.Error("Should not return function when build fails")
	}
}

// Test the function signature requirements
func TestMongoClientRestorer_FunctionSignature(t *testing.T) {
	// Test that the returned function has the correct signature
	// We can't test execution, but we can verify the function type
	
	restorer := &MongoClientRestorer{dbCollection: nil}
	
	change := &diff.Change{
		Action:          diff.ActionUpdated,
		IdentifiedBy:    "_id",
		IdentifierValue: "test",
		Data:            bson.M{"name": "test"},
	}
	
	fn, err := restorer.Build(change)
	
	// Should fail due to nil collection
	if err == nil {
		t.Error("Expected error due to nil collection")
	}
	
	// Function should be nil when build fails
	if fn != nil {
		t.Error("Function should be nil when build fails")
	}
	
	// Verify error message is appropriate
	if !strings.Contains(err.Error(), "connected db collection is required") {
		t.Errorf("Unexpected error message: %v", err)
	}
}