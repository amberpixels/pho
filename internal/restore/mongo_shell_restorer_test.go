package restore

import (
	"errors"
	"strings"
	"testing"

	"pho/internal/diff"

	"go.mongodb.org/mongo-driver/bson"
)

func TestNewMongoShellRestorer(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
	}{
		{
			name:           "simple collection name",
			collectionName: "users",
		},
		{
			name:           "collection with underscores",
			collectionName: "user_profiles",
		},
		{
			name:           "collection with dots",
			collectionName: "analytics.events",
		},
		{
			name:           "empty collection name",
			collectionName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restorer := NewMongoShellRestorer(tt.collectionName)

			if restorer == nil {
				t.Fatal("NewMongoShellRestorer() returned nil")
			}

			if restorer.collectionName != tt.collectionName {
				t.Errorf("collectionName = %v, want %v", restorer.collectionName, tt.collectionName)
			}
		})
	}
}

func TestMongoShellRestorer_Build_ValidationErrors(t *testing.T) {
	restorer := NewMongoShellRestorer("testcoll")

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
			name:    "nil change",
			change:  nil,
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

func TestMongoShellRestorer_Build_UpdateAction(t *testing.T) {
	restorer := NewMongoShellRestorer("users")

	tests := []struct {
		name          string
		change        *diff.Change
		wantErr       bool
		wantContains  []string
		errorContains string
	}{
		{
			name: "update with data",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "_id",
				IdentifierValue: "12345",
				Data: bson.M{
					"_id":  "12345",
					"name": "John Doe",
					"age":  30,
				},
			},
			wantErr: false,
			wantContains: []string{
				"db.getCollection(\"users\").updateOne(",
				"_id:12345",
				"$set:",
				"name",
				"John Doe",
			},
		},
		{
			name: "update without data",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "_id",
				IdentifierValue: "12345",
				Data:            nil,
			},
			wantErr:       true,
			errorContains: "updated action requires a doc",
		},
		{
			name: "update with string identifier value",
			change: &diff.Change{
				Action:          diff.ActionUpdated,
				IdentifiedBy:    "email",
				IdentifierValue: "user@example.com",
				Data: bson.M{
					"email": "user@example.com",
					"name":  "Jane Doe",
				},
			},
			wantErr: false,
			wantContains: []string{
				"db.getCollection(\"users\").updateOne(",
				"email:user@example.com",
				"$set:",
				"name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := restorer.Build(tt.change)

			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errorContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errorContains)) {
					t.Errorf("Build() error = %v, want error containing %v", err, tt.errorContains)
				}
				return
			}

			// Check that the result contains expected strings
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Build() result = %v, want to contain %v", result, want)
				}
			}

			// Ensure _id field is excluded from $set operation (it shouldn't be updated)
			if tt.change.Data != nil {
				if _, hasID := tt.change.Data["_id"]; hasID && tt.change.IdentifiedBy == "_id" {
					// The original data should still have _id, but the command shouldn't include it in $set
					lines := strings.Split(result, ":")
					setIndex := -1
					for i, line := range lines {
						if strings.Contains(line, "$set") {
							setIndex = i
							break
						}
					}
					if setIndex != -1 && setIndex+1 < len(lines) {
						setContent := strings.Join(lines[setIndex+1:], ":")
						if strings.Contains(setContent, "\"_id\"") || strings.Contains(setContent, "_id:") {
							t.Error("Update command should not include _id field in $set operation")
						}
					}
				}
			}
		})
	}
}

func TestMongoShellRestorer_Build_AddAction(t *testing.T) {
	restorer := NewMongoShellRestorer("products")

	tests := []struct {
		name         string
		change       *diff.Change
		wantErr      bool
		wantContains []string
	}{
		{
			name: "add with data",
			change: &diff.Change{
				Action:          diff.ActionAdded,
				IdentifiedBy:    "_id",
				IdentifierValue: "12345",
				Data: bson.M{
					"_id":   "12345",
					"name":  "Product A",
					"price": 99.99,
				},
			},
			wantErr: false,
			wantContains: []string{
				"db.getCollection(\"products\").insertOne(",
				"name",
				"Product A",
				"price",
			},
		},
		{
			name: "add with nil data",
			change: &diff.Change{
				Action:          diff.ActionAdded,
				IdentifiedBy:    "_id",
				IdentifierValue: "12345",
				Data:            nil,
			},
			wantErr: false,
			wantContains: []string{
				"db.getCollection(\"products\").insertOne(",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := restorer.Build(tt.change)

			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				for _, want := range tt.wantContains {
					if !strings.Contains(result, want) {
						t.Errorf("Build() result = %v, want to contain %v", result, want)
					}
				}

				// Check that it ends with semicolon
				if !strings.HasSuffix(result, ");") {
					t.Errorf("Build() result should end with ');', got %v", result)
				}
			}
		})
	}
}

func TestMongoShellRestorer_Build_DeleteAction(t *testing.T) {
	restorer := NewMongoShellRestorer("logs")

	tests := []struct {
		name         string
		change       *diff.Change
		wantContains []string
	}{
		{
			name: "delete by _id",
			change: &diff.Change{
				Action:          diff.ActionDeleted,
				IdentifiedBy:    "_id",
				IdentifierValue: "12345",
			},
			wantContains: []string{
				"db.getCollection(\"logs\").remove(",
				"\"_id\":12345",
			},
		},
		{
			name: "delete by email",
			change: &diff.Change{
				Action:          diff.ActionDeleted,
				IdentifiedBy:    "email",
				IdentifierValue: "user@example.com",
			},
			wantContains: []string{
				"db.getCollection(\"logs\").remove(",
				"\"email\":user@example.com",
			},
		},
		{
			name: "delete by numeric ID",
			change: &diff.Change{
				Action:          diff.ActionDeleted,
				IdentifiedBy:    "user_id",
				IdentifierValue: 12345,
			},
			wantContains: []string{
				"db.getCollection(\"logs\").remove(",
				"\"user_id\":12345",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := restorer.Build(tt.change)

			if err != nil {
				t.Errorf("Build() unexpected error: %v", err)
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Build() result = %v, want to contain %v", result, want)
				}
			}

			// Check that it ends with semicolon
			if !strings.HasSuffix(result, ");") {
				t.Errorf("Build() result should end with ');', got %v", result)
			}
		})
	}
}

func TestMongoShellRestorer_Build_NoopAction(t *testing.T) {
	restorer := NewMongoShellRestorer("collection")

	change := &diff.Change{
		Action:          diff.ActionNoop,
		IdentifiedBy:    "_id",
		IdentifierValue: "12345",
	}

	result, err := restorer.Build(change)

	if !errors.Is(err, ErrNoop) {
		t.Errorf("Build() error = %v, want ErrNoop", err)
	}

	if result != "" {
		t.Errorf("Build() result = %v, want empty string for noop", result)
	}
}

func TestMongoShellRestorer_Build_InvalidAction(t *testing.T) {
	restorer := NewMongoShellRestorer("collection")

	change := &diff.Change{
		Action:          diff.Action(99), // Invalid action
		IdentifiedBy:    "_id",
		IdentifierValue: "12345",
	}

	result, err := restorer.Build(change)

	if err == nil {
		t.Error("Build() expected error for invalid action, got nil")
	}

	if !strings.Contains(err.Error(), "invalid action type") {
		t.Errorf("Build() error = %v, want error containing 'invalid action type'", err)
	}

	if result != "" {
		t.Errorf("Build() result = %v, want empty string for error", result)
	}
}

func TestMongoShellRestorer_Build_CollectionNameEscaping(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		expectedInCmd  string
	}{
		{
			name:           "simple name",
			collectionName: "users",
			expectedInCmd:  "\"users\"",
		},
		{
			name:           "name with dots",
			collectionName: "analytics.events",
			expectedInCmd:  "\"analytics.events\"",
		},
		{
			name:           "name with special characters",
			collectionName: "user-profiles_2024",
			expectedInCmd:  "\"user-profiles_2024\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restorer := NewMongoShellRestorer(tt.collectionName)

			change := &diff.Change{
				Action:          diff.ActionDeleted,
				IdentifiedBy:    "_id",
				IdentifierValue: "test",
			}

			result, err := restorer.Build(change)
			if err != nil {
				t.Errorf("Build() unexpected error: %v", err)
				return
			}

			if !strings.Contains(result, tt.expectedInCmd) {
				t.Errorf("Build() result = %v, want to contain %v", result, tt.expectedInCmd)
			}
		})
	}
}

func TestMongoShellRestorer_Build_DataCloning(t *testing.T) {
	// Test that data cloning works and doesn't mutate original
	restorer := NewMongoShellRestorer("test")

	originalData := bson.M{
		"_id":  "12345",
		"name": "original",
		"tags": []string{"a", "b"},
	}

	change := &diff.Change{
		Action:          diff.ActionUpdated,
		IdentifiedBy:    "_id",
		IdentifierValue: "12345",
		Data:            originalData,
	}

	_, err := restorer.Build(change)
	if err != nil {
		t.Errorf("Build() unexpected error: %v", err)
		return
	}

	// Check that original data wasn't modified
	if originalData["name"] != "original" {
		t.Error("Original data was modified during build")
	}

	if _, exists := originalData["_id"]; !exists {
		t.Error("Original data should still contain _id field")
	}
}

func TestMongoShellRestorer_Build_ComplexData(t *testing.T) {
	restorer := NewMongoShellRestorer("complex")

	change := &diff.Change{
		Action:          diff.ActionAdded,
		IdentifiedBy:    "_id",
		IdentifierValue: "12345",
		Data: bson.M{
			"_id":    "12345",
			"nested": bson.M{"field": "value"},
			"array":  []any{1, "two", true},
			"number": 42.5,
			"bool":   false,
		},
	}

	result, err := restorer.Build(change)
	if err != nil {
		t.Errorf("Build() unexpected error: %v", err)
		return
	}

	// Should contain the basic structure
	if !strings.Contains(result, "db.getCollection(\"complex\").insertOne(") {
		t.Errorf("Build() result should contain insertOne command")
	}

	// Should be valid shell command syntax
	if !strings.HasSuffix(result, ");") {
		t.Errorf("Build() result should end with ');'")
	}
}

func TestMongoShellRestorer_Build_EmptyData(t *testing.T) {
	restorer := NewMongoShellRestorer("test")

	change := &diff.Change{
		Action:          diff.ActionAdded,
		IdentifiedBy:    "_id",
		IdentifierValue: "12345",
		Data:            bson.M{},
	}

	result, err := restorer.Build(change)
	if err != nil {
		t.Errorf("Build() unexpected error: %v", err)
		return
	}

	if !strings.Contains(result, "insertOne(") {
		t.Error("Build() should generate insertOne command for empty data")
	}
}
