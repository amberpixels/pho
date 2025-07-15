package pho

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// parseQuery parses query string into bson.M.
func parseQuery(queryStr string) (bson.M, error) {
	var query bson.M
	if err := json.Unmarshal([]byte(queryStr), &query); err != nil {
		return nil, fmt.Errorf("error parsing query: %w", err)
	}

	return query, nil
}

// parseSort parses sort string into bson.D.
func parseSort(sortStr string) bson.D {
	var sort bson.D

	// Parse ready-to-use sort like `{"created_at":-1}`
	if strings.HasPrefix(sortStr, "{") {
		err := json.Unmarshal([]byte(sortStr), &sort)
		if err != nil {
			// Return empty sort instead of fatal error
			return bson.D{}
		}
		return sort
	}

	// Parse field names prefixed with - or +
	fields := strings.Split(sortStr, ",")
	for _, field := range fields {
		direction := 1
		if strings.HasPrefix(field, "-") {
			direction = -1
			field = strings.TrimPrefix(field, "-")
		} else if strings.HasPrefix(field, "+") {
			field = strings.TrimPrefix(field, "+")
		}
		sort = append(sort, bson.E{Key: field, Value: direction})
	}
	return sort
}

// parseProjection parses projection string into bson.D.
// Supports formats:
// - "field1,field2,field3" - include specific fields
// - "-field1,-field2" - exclude specific fields (cannot mix include/exclude)
// - JSON format: '{"field1": 1, "field2": 0}'.
func parseProjection(in string) bson.D {
	if in == "" {
		return nil
	}

	// Try to parse as JSON first
	if strings.HasPrefix(in, "{") && strings.HasSuffix(in, "}") {
		var projection bson.M
		if err := bson.UnmarshalExtJSON([]byte(in), true, &projection); err == nil {
			var result bson.D
			for key, value := range projection {
				result = append(result, bson.E{Key: key, Value: value})
			}
			return result
		}
	}

	// Parse comma-separated fields
	fields := strings.Split(in, ",")
	var projection bson.D

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		value := 1 // include by default
		if strings.HasPrefix(field, "-") {
			value = 0 // exclude
			field = strings.TrimPrefix(field, "-")
		} else if strings.HasPrefix(field, "+") {
			field = strings.TrimPrefix(field, "+")
		}
		projection = append(projection, bson.E{Key: field, Value: value})
	}
	return projection
}
