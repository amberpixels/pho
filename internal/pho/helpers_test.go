package pho

import (
	"reflect"
	"testing"

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
			result, err := parseQuery(tt.queryStr)

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
			result := parseSort(tt.sortStr)

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
			expected: bson.D{{Key: "", Value: 1}},
		},
		{
			name:     "single field include",
			projStr:  "name",
			expected: bson.D{{Key: "name", Value: 1}},
		},
		{
			name:     "single field exclude",
			projStr:  "-_id",
			expected: bson.D{{Key: "_id", Value: -1}},
		},
		{
			name:    "multiple fields",
			projStr: "name,email,-_id",
			expected: bson.D{
				{Key: "name", Value: 1},
				{Key: "email", Value: 1},
				{Key: "_id", Value: -1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseProjection(tt.projStr)

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
			result := parseSort(tt.sortStr)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseSort() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test that parseProjection is indeed using parseSort internally.
func TestParseProjection_UsesParseSort(t *testing.T) {
	input := "name,-_id"

	projResult := parseProjection(input)
	sortResult := parseSort(input)

	if !reflect.DeepEqual(projResult, sortResult) {
		t.Errorf("parseProjection() should use parseSort() internally, but results differ")
		t.Errorf("parseProjection() = %v", projResult)
		t.Errorf("parseSort() = %v", sortResult)
	}
}
