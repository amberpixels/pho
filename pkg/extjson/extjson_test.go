package extjson_test

import (
	"go.mongodb.org/mongo-driver/bson"
	"pho/pkg/extjson"
	"testing"
)

func TestMarshaller_Marshal_OnSingleObject(t *testing.T) {
	mrshlr := extjson.NewMarshaller(true)

	testData := bson.M{
		"_id":  "1",
		"foo":  "bar",
		"name": "Bar",
		"xyz":  true,
		"yo":   "1",
	}

	var stable []byte
	for i := 0; i < 10; i++ {
		got, err := mrshlr.Marshal(testData)
		if err != nil {
			t.Errorf("marshal expects to succeed but failed: %s:", err)
			return
		}

		if len(stable) == 0 {
			stable = got
			continue
		}

		if string(stable) != string(got) {
			t.Errorf("Marshalled result is not stable. %s != %s", string(stable), string(got))
		}
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

	_, err := extjson.NewMarshaller(true).Marshal(testData)
	if err == nil {
		t.Errorf("marshal expects to fail yet as not supported")
		return
	}
}
