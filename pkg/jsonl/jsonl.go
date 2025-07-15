package jsonl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// NewDecoder returns a jsonl decoder
// that actually is a simple json.Decoder with a middleware for cleaning up comments.
func NewDecoder(r io.Reader) *json.Decoder {
	return json.NewDecoder(NewJSONCommentsCleaner(r))
}

func DecodeAll[T any](r io.Reader) ([]T, error) {
	decoder := NewDecoder(r)

	var results []T
	for {
		var result T
		err := decoder.Decode(&result)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode a result: %w", err)
		}

		results = append(results, result)
	}

	return results, nil
}
