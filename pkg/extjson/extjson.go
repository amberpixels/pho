// Package extjson provides an advanced way of marhsalling/unmarshalling for MongoDB's Extended JSON
// Docs:
// ExtJSON (v1): https://www.mongodb.com/docs/manual/reference/mongodb-extended-json-v1/
// ExtJSON (v2): https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/
package extjson

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExtJSONMode represents the different ExtJSON formatting modes
type ExtJSONMode string

const (
	Canonical ExtJSONMode = "canonical"
	Relaxed   ExtJSONMode = "relaxed"
	Shell     ExtJSONMode = "shell"
)

type Marshaller struct {
	mode       ExtJSONMode
	canonical  bool
	escapeHTML bool
	compact    bool

	prefix string
	indent string
}

func NewMarshaller(mode ExtJSONMode) *Marshaller {
	canonical := mode == Canonical
	return &Marshaller{
		mode:      mode,
		canonical: canonical,
	}
}

func NewCanonicalMarshaller() *Marshaller {
	return NewMarshaller(Canonical)
}

func NewRelaxedMarshaller() *Marshaller {
	return NewMarshaller(Relaxed)
}

func NewShellMarshaller() *Marshaller {
	return NewMarshaller(Shell)
}

func (m *Marshaller) WithIndent(v string) *Marshaller { m.indent = v; return m }

func (m *Marshaller) WithCompact(compact bool) *Marshaller { m.compact = compact; return m }

// Marshal provides a stable marshalling across all ExtJSON modes
// "stable" here means that resulting []byte will always be the same (order of keys inside won't change)
func (m *Marshaller) Marshal(result any) ([]byte, error) {
	// For better error handling, let's detect when result is a slice
	// As bson.MarshalExtJson can only handle single objects
	// TODO(1): handle it automagically via loop here
	{
		t := reflect.TypeOf(result)
		k := reflect.TypeOf(result).Kind()
		if k == reflect.Slice || k == reflect.Ptr && t.Elem().Kind() == reflect.Slice {
			return nil, fmt.Errorf("can't marshal array yet")
		}
	}

	// Handle Shell mode separately as it's not valid JSON
	if m.mode == Shell {
		return m.marshalShellExtJSON(result)
	}

	// Handle Canonical and Relaxed modes with stable marshalling
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
	// And as ExtJSON (v2) is a valid json - let's simply make a round-trip marshalling
	// TODO(2): rewrite so it's a efficient solution

	var temp any
	if err = json.Unmarshal(marshalled, &temp); err != nil {
		return nil, fmt.Errorf("stabilizing marshalled bson failed on json unmarshalling: %w", err)
	}

	// Apply formatting based on compact setting
	if m.compact || (m.indent == "" && m.prefix == "") {
		if marshalled, err = json.Marshal(temp); err != nil {
			return nil, fmt.Errorf("stabilizing marshalled bson failed on json marshalling back: %w", err)
		}
	} else {
		indentStr := m.indent
		if indentStr == "" {
			indentStr = " " // Default single space indent
		}
		if marshalled, err = json.MarshalIndent(temp, m.prefix, indentStr); err != nil {
			return nil, fmt.Errorf("stabilizing marshalled bson failed on json marshalling back: %w", err)
		}
	}

	return marshalled, nil
}

// marshalShellExtJSON converts BSON documents to MongoDB Shell ExtJSON v1 format
// This format uses constructors like ObjectId(), ISODate(), NumberLong() etc.
func (m *Marshaller) marshalShellExtJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	err := m.marshalShellValue(v, &buf, 0)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// marshalShellValue recursively marshals values to Shell format
func (m *Marshaller) marshalShellValue(v any, buf *bytes.Buffer, indent int) error {
	if v == nil {
		buf.WriteString("null")
		return nil
	}

	switch val := v.(type) {
	case primitive.ObjectID:
		buf.WriteString(`ObjectId("`)
		buf.WriteString(val.Hex())
		buf.WriteString(`")`)

	case primitive.DateTime:
		t := val.Time()
		buf.WriteString(`ISODate("`)
		buf.WriteString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
		buf.WriteString(`")`)

	case time.Time:
		buf.WriteString(`ISODate("`)
		buf.WriteString(val.UTC().Format("2006-01-02T15:04:05.000Z"))
		buf.WriteString(`")`)

	case int64:
		buf.WriteString(`NumberLong("`)
		buf.WriteString(strconv.FormatInt(val, 10))
		buf.WriteString(`")`)

	case int32:
		buf.WriteString(`NumberInt("`)
		buf.WriteString(strconv.FormatInt(int64(val), 10))
		buf.WriteString(`")`)

	case float64:
		buf.WriteString(strconv.FormatFloat(val, 'g', -1, 64))

	case string:
		buf.WriteString(`"`)
		buf.WriteString(strings.ReplaceAll(strings.ReplaceAll(val, `\`, `\\`), `"`, `\"`))
		buf.WriteString(`"`)

	case bool:
		if val {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}

	case primitive.Binary:
		buf.WriteString(`BinData(`)
		buf.WriteString(strconv.Itoa(int(val.Subtype)))
		buf.WriteString(`, "`)
		buf.WriteString(base64.StdEncoding.EncodeToString(val.Data))
		buf.WriteString(`")`)

	case primitive.Regex:
		buf.WriteString(`/`)
		buf.WriteString(val.Pattern)
		buf.WriteString(`/`)
		buf.WriteString(val.Options)

	case bson.M:
		buf.WriteString("{")
		if !m.compact {
			buf.WriteString("\n")
		}
		indentStr := strings.Repeat("  ", indent+1)
		first := true
		for k, v := range val {
			if !first {
				buf.WriteString(",")
				if !m.compact {
					buf.WriteString("\n")
				}
			}
			first = false
			if !m.compact {
				buf.WriteString(indentStr)
			}
			buf.WriteString(`"`)
			buf.WriteString(k)
			buf.WriteString(`"`)
			if m.compact {
				buf.WriteString(":")
			} else {
				buf.WriteString(" : ")
			}
			err := m.marshalShellValue(v, buf, indent+1)
			if err != nil {
				return err
			}
		}
		if !m.compact {
			buf.WriteString("\n")
			buf.WriteString(strings.Repeat("  ", indent))
		}
		buf.WriteString("}")

	case []any:
		buf.WriteString("[")
		if !m.compact {
			buf.WriteString("\n")
		}
		indentStr := strings.Repeat("  ", indent+1)
		for i, item := range val {
			if i > 0 {
				buf.WriteString(",")
				if !m.compact {
					buf.WriteString("\n")
				}
			}
			if !m.compact {
				buf.WriteString(indentStr)
			}
			err := m.marshalShellValue(item, buf, indent+1)
			if err != nil {
				return err
			}
		}
		if !m.compact {
			buf.WriteString("\n")
			buf.WriteString(strings.Repeat("  ", indent))
		}
		buf.WriteString("]")

	default:
		// Handle interface{} and reflect.Value for more complex types
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Map:
			buf.WriteString("{")
			if !m.compact {
				buf.WriteString("\n")
			}
			indentStr := strings.Repeat("  ", indent+1)
			keys := rv.MapKeys()
			for i, key := range keys {
				if i > 0 {
					buf.WriteString(",")
					if !m.compact {
						buf.WriteString("\n")
					}
				}
				if !m.compact {
					buf.WriteString(indentStr)
				}
				buf.WriteString(`"`)
				buf.WriteString(fmt.Sprintf("%v", key.Interface()))
				buf.WriteString(`"`)
				if m.compact {
					buf.WriteString(":")
				} else {
					buf.WriteString(" : ")
				}
				err := m.marshalShellValue(rv.MapIndex(key).Interface(), buf, indent+1)
				if err != nil {
					return err
				}
			}
			if !m.compact {
				buf.WriteString("\n")
				buf.WriteString(strings.Repeat("  ", indent))
			}
			buf.WriteString("}")

		case reflect.Slice, reflect.Array:
			buf.WriteString("[")
			if !m.compact {
				buf.WriteString("\n")
			}
			indentStr := strings.Repeat("  ", indent+1)
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					buf.WriteString(",")
					if !m.compact {
						buf.WriteString("\n")
					}
				}
				if !m.compact {
					buf.WriteString(indentStr)
				}
				err := m.marshalShellValue(rv.Index(i).Interface(), buf, indent+1)
				if err != nil {
					return err
				}
			}
			if !m.compact {
				buf.WriteString("\n")
				buf.WriteString(strings.Repeat("  ", indent))
			}
			buf.WriteString("]")

		default:
			// Fallback to string representation
			buf.WriteString(fmt.Sprintf("%v", v))
		}
	}

	return nil
}
