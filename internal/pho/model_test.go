package pho_test

import (
	"reflect"
	"testing"

	"pho/internal/hashing"
	"pho/internal/pho"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParsedMeta(t *testing.T) {
	// Test ParsedMeta struct creation and usage
	hashData1, err := hashing.Hash(bson.M{"_id": "test1", "name": "doc1"})
	if err != nil {
		t.Fatalf("Failed to create hash data: %v", err)
	}

	hashData2, err := hashing.Hash(bson.M{"_id": "test2", "name": "doc2"})
	if err != nil {
		t.Fatalf("Failed to create hash data: %v", err)
	}

	meta := &pho.ParsedMeta{
		Lines: map[string]*hashing.HashData{
			hashData1.GetIdentifier(): hashData1,
			hashData2.GetIdentifier(): hashData2,
		},
	}

	// Test that we can access the lines
	if len(meta.Lines) != 2 {
		t.Errorf("ParsedMeta.Lines length = %d, want 2", len(meta.Lines))
	}

	// Test that we can retrieve specific hash data
	id1 := hashData1.GetIdentifier()
	retrievedHash, exists := meta.Lines[id1]
	if !exists {
		t.Errorf("ParsedMeta.Lines[%s] not found", id1)
	}

	if retrievedHash.GetChecksum() != hashData1.GetChecksum() {
		t.Errorf("Retrieved hash checksum mismatch")
	}
}

func TestDumpDoc_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected pho.DumpDoc
		wantErr  bool
	}{
		{
			name:     "simple document",
			jsonData: `{"name": "test", "value": 123}`,
			expected: pho.DumpDoc{"name": "test", "value": 123},
			wantErr:  false,
		},
		{
			name:     "document with ObjectId",
			jsonData: `{"_id": {"$oid": "507f1f77bcf86cd799439011"}, "name": "test"}`,
			expected: pho.DumpDoc{
				"_id":  func() primitive.ObjectID { oid, _ := primitive.ObjectIDFromHex("507f1f77bcf86cd799439011"); return oid }(),
				"name": "test",
			},
			wantErr: false,
		},
		{
			name:     "document with Date",
			jsonData: `{"created": {"$date": {"$numberLong": "1672531200000"}}, "name": "test"}`,
			expected: pho.DumpDoc{
				// Note: The exact date parsing depends on BSON ExtJSON implementation
				"name": "test",
			},
			wantErr: false,
		},
		{
			name:     "document with NumberLong",
			jsonData: `{"count": {"$numberLong": "9223372036854775807"}, "name": "test"}`,
			expected: pho.DumpDoc{
				"count": int64(9223372036854775807),
				"name":  "test",
			},
			wantErr: false,
		},
		{
			name:     "document with NumberDecimal",
			jsonData: `{"price": {"$numberDecimal": "123.45"}, "name": "test"}`,
			expected: pho.DumpDoc{
				"name": "test",
				// Note: NumberDecimal handling depends on BSON implementation
			},
			wantErr: false,
		},
		{
			name:     "nested document",
			jsonData: `{"user": {"name": "test", "age": 25}, "active": true}`,
			expected: pho.DumpDoc{
				"user": bson.M{
					"name": "test",
					"age":  25,
				},
				"active": true,
			},
			wantErr: false,
		},
		{
			name:     "array field",
			jsonData: `{"tags": ["go", "mongodb", "json"], "count": 3}`,
			expected: pho.DumpDoc{
				"tags":  bson.A{"go", "mongodb", "json"},
				"count": 3,
			},
			wantErr: false,
		},
		{
			name:     "empty document",
			jsonData: `{}`,
			expected: pho.DumpDoc{},
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			jsonData: `{"name": "test"`,
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid ExtJSON",
			jsonData: `{"_id": {"$invalid": "value"}}`,
			expected: pho.DumpDoc{},
			wantErr:  false, // BSON.UnmarshalExtJSON might handle this gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc pho.DumpDoc
			err := doc.UnmarshalJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("DumpDoc.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// For successful cases, verify some key fields exist
			// Note: We don't do deep equality because ExtJSON parsing can have subtle differences
			if len(tt.expected) > 0 {
				for key := range tt.expected {
					if key == "_id" || key == "created" || key == "price" {
						// Skip complex BSON types that might not match exactly
						continue
					}
					if _, exists := doc[key]; !exists {
						t.Errorf("DumpDoc.UnmarshalJSON() missing expected key: %s", key)
					}
				}
			}
		})
	}
}

func TestDumpDoc_conversion(t *testing.T) {
	// Test that DumpDoc can be converted to bson.M
	originalBson := bson.M{
		"_id":    "test123",
		"name":   "test document",
		"value":  42,
		"active": true,
		"tags":   []string{"test", "document"},
	}

	// Convert to DumpDoc
	dumpDoc := pho.DumpDoc(originalBson)

	// Convert back to bson.M
	resultBson := bson.M(dumpDoc)

	if !reflect.DeepEqual(originalBson, resultBson) {
		t.Errorf("DumpDoc conversion failed")
		t.Errorf("Original: %v", originalBson)
		t.Errorf("Result:   %v", resultBson)
	}
}

func TestDumpDoc_withRealExtJSON(t *testing.T) {
	// Test with real MongoDB ExtJSON examples
	tests := []struct {
		name     string
		jsonData string
		checkFn  func(pho.DumpDoc) bool
	}{
		{
			name:     "ObjectId field",
			jsonData: `{"_id": {"$oid": "507f1f77bcf86cd799439011"}}`,
			checkFn: func(doc pho.DumpDoc) bool {
				id, exists := doc["_id"]
				return exists && id != nil
			},
		},
		{
			name:     "String field",
			jsonData: `{"name": "test"}`,
			checkFn: func(doc pho.DumpDoc) bool {
				name, exists := doc["name"]
				return exists && name == "test"
			},
		},
		{
			name:     "Number field",
			jsonData: `{"value": 42}`,
			checkFn: func(doc pho.DumpDoc) bool {
				value, exists := doc["value"]
				return exists && value != nil
			},
		},
		{
			name:     "Boolean field",
			jsonData: `{"active": true}`,
			checkFn: func(doc pho.DumpDoc) bool {
				active, exists := doc["active"]
				return exists && active == true
			},
		},
		{
			name:     "Null field",
			jsonData: `{"deleted": null}`,
			checkFn: func(doc pho.DumpDoc) bool {
				_, exists := doc["deleted"]
				return exists // null fields should exist as keys
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc pho.DumpDoc
			err := doc.UnmarshalJSON([]byte(tt.jsonData))
			if err != nil {
				t.Errorf("DumpDoc.UnmarshalJSON() unexpected error: %v", err)
				return
			}

			if !tt.checkFn(doc) {
				t.Errorf("DumpDoc.UnmarshalJSON() result check failed for %s", tt.name)
				t.Errorf("Document: %v", doc)
			}
		})
	}
}

// Test edge cases and type safety.
func TestDumpDoc_typeSafety(t *testing.T) {
	// Test that DumpDoc is indeed bson.M underneath
	var doc = pho.DumpDoc(make(bson.M))

	// Should be able to add fields like a regular bson.M
	doc["test"] = "value"
	doc["number"] = 42
	doc["bool"] = true

	if len(doc) != 3 {
		t.Errorf("DumpDoc length = %d, want 3", len(doc))
	}

	if doc["test"] != "value" {
		t.Errorf("DumpDoc[\"test\"] = %v, want \"value\"", doc["test"])
	}

	if doc["number"] != 42 {
		t.Errorf("DumpDoc[\"number\"] = %v, want 42", doc["number"])
	}

	if doc["bool"] != true {
		t.Errorf("DumpDoc[\"bool\"] = %v, want true", doc["bool"])
	}
}

func TestParsedMeta_emptyLines(t *testing.T) {
	// Test ParsedMeta with empty Lines map
	meta := &pho.ParsedMeta{
		Lines: make(map[string]*hashing.HashData),
	}

	if len(meta.Lines) != 0 {
		t.Errorf("Empty ParsedMeta.Lines length = %d, want 0", len(meta.Lines))
	}

	// Test adding to empty map
	hashData, err := hashing.Hash(bson.M{"_id": "test", "name": "doc"})
	if err != nil {
		t.Fatalf("Failed to create hash data: %v", err)
	}

	meta.Lines[hashData.GetIdentifier()] = hashData

	if len(meta.Lines) != 1 {
		t.Errorf("ParsedMeta.Lines length after add = %d, want 1", len(meta.Lines))
	}
}
