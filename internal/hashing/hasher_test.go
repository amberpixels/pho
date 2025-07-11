package hashing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, hashData)

			// Verify hash components
			assert.NotEmpty(t, hashData.GetChecksum())
			assert.NotEmpty(t, hashData.GetIdentifier())

			// Verify identifier parsing
			identifiedBy, identifierValue := hashData.GetIdentifierParts()
			assert.Equal(t, "_id", identifiedBy)
			assert.NotNil(t, identifierValue)
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
	require.NoError(t, err)

	str := hashData.String()
	assert.NotEmpty(t, str)

	// Verify format: should contain separator
	assert.Contains(t, str, ChecksumSeparator)
	assert.Contains(t, str, IdentifierSeparator)
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
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, hashData)

			// Verify round-trip consistency
			reconstructed := hashData.String()
			assert.Equal(t, tt.input, reconstructed)
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
	require.NoError(t, err)

	hash2, err := Hash(doc)
	require.NoError(t, err)

	assert.Equal(t, hash1.String(), hash2.String())
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
	require.NoError(t, err)

	hash2, err := Hash(doc2)
	require.NoError(t, err)

	assert.NotEqual(t, hash1.GetChecksum(), hash2.GetChecksum())
	assert.Equal(t, hash1.GetIdentifier(), hash2.GetIdentifier())
}
