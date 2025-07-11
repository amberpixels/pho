package hashing

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHash(t *testing.T) {
	tests := []struct {
		name    string
		doc     bson.M
		wantErr bool
	}{
		{
			name: "document with ObjectID",
			doc: bson.M{
				"_id":  primitive.NewObjectID(),
				"name": "test",
				"data": map[string]interface{}{"nested": "value"},
			},
			wantErr: false,
		},
		{
			name: "document with string ID",
			doc: bson.M{
				"_id":    "string-id",
				"value":  42,
				"active": true,
			},
			wantErr: false,
		},
		{
			name: "document without _id",
			doc: bson.M{
				"name": "no-id",
				"data": "value",
			},
			wantErr: true,
		},
		{
			name:    "empty document",
			doc:     bson.M{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashData, err := Hash(tt.doc)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Hash() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Hash() unexpected error: %v", err)
				return
			}

			if hashData == nil {
				t.Errorf("Hash() returned nil hashData")
				return
			}

			// Verify hash components
			if hashData.GetChecksum() == "" {
				t.Errorf("Hash() returned empty checksum")
			}

			if hashData.GetIdentifier() == "" {
				t.Errorf("Hash() returned empty identifier")
			}

			// Verify identifier parsing
			identifiedBy, identifierValue := hashData.GetIdentifierParts()
			if identifiedBy != "_id" {
				t.Errorf("Hash() expected identifiedBy '_id', got '%s'", identifiedBy)
			}

			if identifierValue == nil {
				t.Errorf("Hash() returned nil identifierValue")
			}
		})
	}
}

func TestHashData_String(t *testing.T) {
	// Create a test document
	doc := bson.M{
		"_id":  "test-id",
		"name": "test document",
	}

	hashData, err := Hash(doc)
	if err != nil {
		t.Fatalf("Hash() failed: %v", err)
	}

	str := hashData.String()
	if str == "" {
		t.Errorf("String() returned empty string")
	}

	// Verify format: should contain separator
	if !containsString(str, ChecksumSeparator) {
		t.Errorf("String() should contain checksum separator '%s'", ChecksumSeparator)
	}

	if !containsString(str, IdentifierSeparator) {
		t.Errorf("String() should contain identifier separator '%s'", IdentifierSeparator)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid hash string with hex ID",
			input:   "_id::507f1f77bcf86cd799439012|abcdef123456",
			wantErr: false,
		},
		{
			name:    "valid hash string with ObjectID format",
			input:   "_id::ObjectID(\"507f1f77bcf86cd799439011\")|abcdef123456",
			wantErr: false,
		},
		{
			name:    "missing checksum separator",
			input:   "_id::test-id",
			wantErr: true,
		},
		{
			name:    "missing identifier separator",
			input:   "_id-test-id|abcdef123456",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid ObjectID hex",
			input:   "_id::ObjectID(\"invalid-hex\")|abcdef123456",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashData, err := Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error: %v", err)
				return
			}

			if hashData == nil {
				t.Errorf("Parse() returned nil hashData")
				return
			}

			// Verify round-trip consistency
			reconstructed := hashData.String()
			if reconstructed != tt.input {
				t.Errorf("Parse() round-trip failed: got %s, want %s", reconstructed, tt.input)
			}
		})
	}
}

func TestHashConsistency(t *testing.T) {
	// Test that the same document produces the same hash
	doc := bson.M{
		"_id":    "consistent-test",
		"field1": "value1",
		"field2": 42,
		"field3": true,
	}

	hash1, err := Hash(doc)
	if err != nil {
		t.Fatalf("First hash failed: %v", err)
	}

	hash2, err := Hash(doc)
	if err != nil {
		t.Fatalf("Second hash failed: %v", err)
	}

	if hash1.String() != hash2.String() {
		t.Errorf("Hash consistency failed: %s != %s", hash1.String(), hash2.String())
	}
}

func TestHashSensitivity(t *testing.T) {
	// Test that different documents produce different hashes
	doc1 := bson.M{
		"_id":   "test-1",
		"value": "original",
	}

	doc2 := bson.M{
		"_id":   "test-1",
		"value": "modified",
	}

	hash1, err := Hash(doc1)
	if err != nil {
		t.Fatalf("First hash failed: %v", err)
	}

	hash2, err := Hash(doc2)
	if err != nil {
		t.Fatalf("Second hash failed: %v", err)
	}

	if hash1.GetChecksum() == hash2.GetChecksum() {
		t.Errorf("Different documents should produce different checksums")
	}

	if hash1.GetIdentifier() != hash2.GetIdentifier() {
		t.Errorf("Same _id should produce same identifier")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] != substr && s[:len(substr)] != substr &&
		findInString(s, substr)
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
