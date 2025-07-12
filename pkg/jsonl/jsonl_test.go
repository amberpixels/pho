package jsonl_test

import (
	"os"
	"testing"

	"pho/pkg/jsonl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonlDecoding(t *testing.T) {
	type Obj struct {
		Name string `json:"name"`
		ID   struct {
			Oid string `json:"$oid"`
		} `json:"_id"`
		V *struct {
			Foo string `json:"foo"`
		} `json:"v,omitempty"`
		Nested []struct {
			V string `json:"v"`
		} `json:"nested,omitempty"`
	}

	for _, filename := range []string{
		"with-comments.compact",
		"with-comments",
		"with-multiline-comments.compact",
		"with-multiline-comments",
		"without-comments.compact",
		"without-comments",
	} {
		file, err := os.Open("testdata/samples." + filename + ".jsonl")
		require.NoError(t, err, "could not read testdata file %s", filename)

		decoded, err := jsonl.DecodeAll[Obj](file)
		require.NoError(t, err, "could not decode all %s", filename)

		assert.Len(t, decoded, 3, "len(decoded) must be 3 (%s)", filename)

		assert.Equal(t, "Sample 9", decoded[0].Name)
		assert.Equal(t, "Sample 8", decoded[1].Name)
		assert.Equal(t, "Sample 7", decoded[2].Name)
	}
}
