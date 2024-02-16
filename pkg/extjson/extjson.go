// Package extjson provides an advanced way of marhsalling/unmarshalling for MongoDB's Extended JSON
// Docs:
// ExtJSON (v1): https://www.mongodb.com/docs/manual/reference/mongodb-extended-json-v1/
// ExtJSON (v2): https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/
package extjson

import (
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
)

type Marshaller struct {
	canonical  bool
	escapeHTML bool

	prefix string
	indent string
}

func NewMarshaller(canonical bool) *Marshaller {
	return &Marshaller{canonical: canonical}
}

func (m *Marshaller) WithIndent(v string) *Marshaller { m.indent = v; return m }

// Marshal provides a stable marshalling
// "stable" here means that resulting []byte will always be the same (order of keys inside won't change)
func (m *Marshaller) Marshal(result any) ([]byte, error) {
	// For better error handling, let's detect when result is a slice
	// As bson.MarshalExtJson can only handle single objects
	// todo(3): handle it automagically via loop here
	{
		t := reflect.TypeOf(result)
		k := reflect.TypeOf(result).Kind()
		if k == reflect.Slice || k == reflect.Ptr && t.Elem().Kind() == reflect.Slice {
			return nil, fmt.Errorf("can't marshal array yet")

		}
	}

	var marshalled []byte
	var err error
	if m.indent != "" || m.prefix != "" {
		marshalled, err = bson.MarshalExtJSONIndent(result, m.canonical, m.escapeHTML, m.prefix, m.indent)
	} else {
		marshalled, err = bson.MarshalExtJSON(result, m.canonical, m.escapeHTML)
	}

	if err != nil {
		return nil, err
	}
	// bson.MarshalExtJson is not stable
	// It's not trivial to rewrite it
	// So for Day One solution we're completely OK with not the most efficient
	// but working solution
	// So, as json.Marshal() does provide a stable marshalling into a map
	// And our ExtJSON (v2) is a valid json - let's simply make a round-trip marshalling
	// todo(1): rewrite so it's a efficient solution
	// todo(2): think of a solution to support ExtJSON (v1) as it's not a valid JSON

	var temp any
	if err = json.Unmarshal(marshalled, &temp); err != nil {
		return nil, fmt.Errorf("stabilizing marshalled bson failed on json unmarshalling: %w", err)
	}
	if m.indent != "" || m.prefix != "" {
		if marshalled, err = json.MarshalIndent(temp, m.prefix, m.indent); err != nil {
			return nil, fmt.Errorf("stabilizing marshalled bson failed on json marshalling back: %w", err)
		}
	} else {
		if marshalled, err = json.Marshal(temp); err != nil {
			return nil, fmt.Errorf("stabilizing marshalled bson failed on json marshalling back: %w", err)
		}
	}

	return marshalled, nil
}
