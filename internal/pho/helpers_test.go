package pho_test

import (
	"reflect"
	"testing"

	"pho/internal/pho"

	"go.mongodb.org/mongo-driver/bson"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name     string
		queryStr string
		expected bson.M
		wantErr  bool
	}{
		{
			name:     "empty query",
			queryStr: "{}",
			expected: bson.M{},
			wantErr:  false,
		},
		{
			name:     "simple field query",
			queryStr: `{"name": "test"}`,
			expected: bson.M{"name": "test"},
			wantErr:  false,
		},
		{
			name:     "numeric field query",
			queryStr: `{"age": 25}`,
			expected: bson.M{"age": 25.0}, // JSON numbers become float64
			wantErr:  false,
		},
		{
			name:     "nested object query",
			queryStr: `{"user": {"name": "test", "active": true}}`,
			expected: bson.M{"user": map[string]any{"name": "test", "active": true}},
			wantErr:  false,
		},
		{
			name:     "array query",
			queryStr: `{"tags": ["go", "mongodb"]}`,
			expected: bson.M{"tags": []any{"go", "mongodb"}},
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			queryStr: `{"name": "test"`,
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "malformed JSON",
			queryStr: `{name: "test"}`,
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "empty string",
			queryStr: "",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pho.ParseQuery(tt.queryStr)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseQuery() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseSort(t *testing.T) {
	tests := []struct {
		name     string
		sortStr  string
		expected bson.D
	}{
		{
			name:     "empty string",
			sortStr:  "",
			expected: bson.D{{Key: "", Value: 1}},
		},
		{
			name:     "single field ascending",
			sortStr:  "name",
			expected: bson.D{{Key: "name", Value: 1}},
		},
		{
			name:     "single field descending",
			sortStr:  "-name",
			expected: bson.D{{Key: "name", Value: -1}},
		},
		{
			name:     "single field explicit ascending",
			sortStr:  "+name",
			expected: bson.D{{Key: "name", Value: 1}},
		},
		{
			name:    "multiple fields",
			sortStr: "name,-age,+status",
			expected: bson.D{
				{Key: "name", Value: 1},
				{Key: "age", Value: -1},
				{Key: "status", Value: 1},
			},
		},
		{
			name:    "complex field names",
			sortStr: "user.name,-user.age",
			expected: bson.D{
				{Key: "user.name", Value: 1},
				{Key: "user.age", Value: -1},
			},
		},
		{
			name:    "field with underscores",
			sortStr: "created_at,-updated_at",
			expected: bson.D{
				{Key: "created_at", Value: 1},
				{Key: "updated_at", Value: -1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pho.ParseSort(tt.sortStr)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseSort() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseProjection(t *testing.T) {
	tests := []struct {
		name     string
		projStr  string
		expected bson.D
	}{
		{
			name:     "empty projection",
			projStr:  "",
			expected: nil,
		},
		{
			name:     "single field include",
			projStr:  "name",
			expected: bson.D{{Key: "name", Value: 1}},
		},
		{
			name:     "single field exclude",
			projStr:  "-_id",
			expected: bson.D{{Key: "_id", Value: 0}},
		},
		{
			name:    "multiple fields",
			projStr: "name,email,-_id",
			expected: bson.D{
				{Key: "name", Value: 1},
				{Key: "email", Value: 1},
				{Key: "_id", Value: 0},
			},
		},
		{
			name:    "complex field names",
			projStr: "user.name,-user.password,+user.email",
			expected: bson.D{
				{Key: "user.name", Value: 1},
				{Key: "user.password", Value: 0},
				{Key: "user.email", Value: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pho.ParseProjection(tt.projStr)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseProjection() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test edge cases for parseSort with complex scenarios.
func TestParseSort_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		sortStr  string
		expected bson.D
	}{
		{
			name:    "only commas",
			sortStr: ",,",
			expected: bson.D{
				{Key: "", Value: 1},
				{Key: "", Value: 1},
				{Key: "", Value: 1},
			},
		},
		{
			name:    "trailing comma",
			sortStr: "name,",
			expected: bson.D{
				{Key: "name", Value: 1},
				{Key: "", Value: 1},
			},
		},
		{
			name:    "leading comma",
			sortStr: ",name",
			expected: bson.D{
				{Key: "", Value: 1},
				{Key: "name", Value: 1},
			},
		},
		{
			name:     "only plus sign",
			sortStr:  "+",
			expected: bson.D{{Key: "", Value: 1}},
		},
		{
			name:     "only minus sign",
			sortStr:  "-",
			expected: bson.D{{Key: "", Value: -1}},
		},
		{
			name:     "multiple plus signs",
			sortStr:  "++name",
			expected: bson.D{{Key: "+name", Value: 1}},
		},
		{
			name:     "multiple minus signs",
			sortStr:  "--name",
			expected: bson.D{{Key: "-name", Value: -1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pho.ParseSort(tt.sortStr)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseSort() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test JSON format projection separately due to map ordering.
func TestParseProjection_JSONFormat(t *testing.T) {
	result := pho.ParseProjection(`{"name": 1, "_id": 0}`)

	// Check that we got 2 elements
	if len(result) != 2 {
		t.Errorf("Expected 2 projection fields, got %d", len(result))
		return
	}

	// Check that both fields are present with correct values
	found := make(map[string]int)
	for _, elem := range result {
		if val, ok := elem.Value.(int); ok {
			found[elem.Key] = val
		} else if val, ok := elem.Value.(int32); ok {
			found[elem.Key] = int(val)
		} else if val, ok := elem.Value.(int64); ok {
			found[elem.Key] = int(val)
		} else {
			t.Errorf("Unexpected value type for %s: %T", elem.Key, elem.Value)
		}
	}

	if found["name"] != 1 || found["_id"] != 0 {
		t.Errorf("Expected name:1, _id:0, got %v", found)
	}
}

// Test edge cases for parseProjection.
func TestParseProjection_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		projStr  string
		expected bson.D
	}{
		{
			name:    "whitespace handling",
			projStr: " name , email , -_id ",
			expected: bson.D{
				{Key: "name", Value: 1},
				{Key: "email", Value: 1},
				{Key: "_id", Value: 0},
			},
		},
		{
			name:     "invalid JSON fallback to comma-separated",
			projStr:  `{"name": invalid}`,
			expected: bson.D{{Key: `{"name": invalid}`, Value: 1}},
		},
		{
			name:    "empty field in list",
			projStr: "name,,email",
			expected: bson.D{
				{Key: "name", Value: 1},
				{Key: "email", Value: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pho.ParseProjection(tt.projStr)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseProjection() = %v, want %v", result, tt.expected)
			}
		})
	}
}
