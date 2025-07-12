package extjson_test

import (
	"pho/pkg/extjson"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMarshaller_Marshal_OnSingleObject(t *testing.T) {
	mrshlr := extjson.NewCanonicalMarshaller()

	testData := bson.M{
		"_id":  "1",
		"foo":  "bar",
		"name": "Bar",
		"xyz":  true,
		"yo":   "1",
	}

	var stable []byte
	for range 10 {
		got, err := mrshlr.Marshal(testData)
		require.NoError(t, err, "marshal expects to succeed but failed")

		if len(stable) == 0 {
			stable = got
			continue
		}

		assert.Equal(t, string(stable), string(got), "Marshalled result is not stable")
	}
}

func TestMarshaller_Marshal_OnArray(t *testing.T) {
	testData := []any{
		bson.M{
			"_id":  "1",
			"foo":  "bar",
			"name": "Bar",
			"xyz":  true,
			"yo":   "1",
		},
	}

	_, err := extjson.NewCanonicalMarshaller().Marshal(testData)
	assert.Error(t, err, "marshal expects to fail yet as not supported")
}
