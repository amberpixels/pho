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
func parseProjection(in string) bson.D {
	// TODO:
	// for now do the same as parseSort, but should be refactored
	return parseSort(in)
}
