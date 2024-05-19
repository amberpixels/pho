package jsonl

import (
	"os"
	"testing"
)

func TestJsonlDecoding(t *testing.T) {

	type Obj struct {
		Name string `json:"name"`
		Id   struct {
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
		if err != nil {
			t.Errorf("could not read testdata file %s: %s", filename, err)
		}

		decoded, err := DecodeAll[Obj](file)
		if err != nil {
			t.Errorf("could not decode all %s: %s", filename, err)
		}

		if len(decoded) != 3 {
			t.Fatalf("len(decoded) must be 3 (%s), got %d", filename, len(decoded))
		}

		if decoded[0].Name != "Sample 9" {
			t.Errorf("decoded[0] name expected to be Sample 9 got %s", decoded[0].Name)
		}
		if decoded[1].Name != "Sample 8" {
			t.Errorf("decoded[0] name expected to be Sample 9 got %s", decoded[0].Name)
		}
		if decoded[2].Name != "Sample 7" {
			t.Errorf("decoded[0] name expected to be Sample 9 got %s", decoded[0].Name)
		}
	}
}
