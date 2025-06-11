package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Renderer struct {
	config *Configuration
}

func NewRenderer(opts ...Option) *Renderer {
	r := &Renderer{}
	r.config = NewConfiguration(opts...)
	return r
}

type Cursor interface {
	Next(context.Context) bool
	Decode(any) error
}

func (r *Renderer) GetConfiguration() *Configuration {
	return r.config
}

func (r *Renderer) FormatLineNumber(lineNumber int) []byte {
	// ignore if no line number needed or Valid JSON is required
	// MinimizedJSON implies valid json as well
	if !r.config.ShowLineNumbers || r.config.AsValidJSON || r.config.MinimizedJSON {
		return nil
	}

	return []byte(fmt.Sprintf("/* %d */\n", lineNumber))
}

func (r *Renderer) FormatResult(result any) ([]byte, error) {
	cfg := r.config

	marshal := func(any) ([]byte, error) { return nil, fmt.Errorf("not specified yet") }
	switch true {
	case cfg.ExtJSONMode == ExtJSONModes.Canonical && !cfg.CompactJSON:
		marshal = func(v any) ([]byte, error) {
			return bson.MarshalExtJSONIndent(result, true, false, "", " ")
		}
	case cfg.ExtJSONMode == ExtJSONModes.Canonical && cfg.CompactJSON:
		marshal = func(v any) ([]byte, error) {
			return bson.MarshalExtJSON(result, true, false)
		}
	case cfg.ExtJSONMode == ExtJSONModes.Relaxed && !cfg.CompactJSON:
		marshal = func(v any) ([]byte, error) {
			return bson.MarshalExtJSONIndent(result, false, false, "", " ")
		}
	case cfg.ExtJSONMode == ExtJSONModes.Relaxed && cfg.CompactJSON:
		marshal = func(v any) ([]byte, error) {
			return bson.MarshalExtJSON(result, false, false)
		}
	case cfg.ExtJSONMode == ExtJSONModes.Shell:
		marshal = func(v any) ([]byte, error) {
			return marshalShellExtJSON(v)
		}
	}

	b, err := marshal(result)
	if err != nil {
		if cfg.IgnoreFailures {
			return nil, nil
		}

		return nil, fmt.Errorf("failed on marshaling result: %w", err)
	}
	if cfg.AsValidJSON {
		b = append(b, []byte(",")...)
	}
	if !cfg.MinimizedJSON {
		b = append(b, []byte("\n")...)
	}

	return b, nil
}

func (r *Renderer) Format(ctx context.Context, cursor Cursor, out io.Writer) error {
	cfg := r.config

	lineNumber := 0
	for cursor.Next(ctx) {
		var result bson.M
		err := cursor.Decode(&result)
		if err != nil {
			if cfg.IgnoreFailures {
				continue
			}

			return fmt.Errorf("failed on decoding line [%d]: %w", lineNumber, err)
		}

		resultBytes, err := r.FormatResult(result)
		if err != nil {
			if cfg.IgnoreFailures {
				continue
			}

			return fmt.Errorf("failed on formatting line [%d]: %w", lineNumber, err)
		}

		if lineNumberBytes := r.FormatLineNumber(lineNumber); lineNumberBytes != nil {
			resultBytes = append(lineNumberBytes, resultBytes...)
		}

		if _, err = out.Write(resultBytes); err != nil {
			if cfg.IgnoreFailures {
				continue
			}

			return fmt.Errorf("failed on writing a line [%d]: %w", lineNumber, err)
		}

		lineNumber++
	}

	return nil
}

// marshalShellExtJSON converts BSON documents to MongoDB Shell ExtJSON v1 format
// This format uses constructors like ObjectId(), ISODate(), NumberLong() etc.
func marshalShellExtJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	err := marshalShellValue(v, &buf, 0)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// marshalShellValue recursively marshals values to Shell format
func marshalShellValue(v any, buf *bytes.Buffer, indent int) error {
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
		buf.WriteString("{\n")
		indentStr := strings.Repeat("  ", indent+1)
		first := true
		for k, v := range val {
			if !first {
				buf.WriteString(",\n")
			}
			first = false
			buf.WriteString(indentStr)
			buf.WriteString(`"`)
			buf.WriteString(k)
			buf.WriteString(`" : `)
			err := marshalShellValue(v, buf, indent+1)
			if err != nil {
				return err
			}
		}
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat("  ", indent))
		buf.WriteString("}")
		
	case []any:
		buf.WriteString("[\n")
		indentStr := strings.Repeat("  ", indent+1)
		for i, item := range val {
			if i > 0 {
				buf.WriteString(",\n")
			}
			buf.WriteString(indentStr)
			err := marshalShellValue(item, buf, indent+1)
			if err != nil {
				return err
			}
		}
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat("  ", indent))
		buf.WriteString("]")
		
	default:
		// Handle interface{} and reflect.Value for more complex types
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Map:
			buf.WriteString("{\n")
			indentStr := strings.Repeat("  ", indent+1)
			keys := rv.MapKeys()
			for i, key := range keys {
				if i > 0 {
					buf.WriteString(",\n")
				}
				buf.WriteString(indentStr)
				buf.WriteString(`"`)
				buf.WriteString(fmt.Sprintf("%v", key.Interface()))
				buf.WriteString(`" : `)
				err := marshalShellValue(rv.MapIndex(key).Interface(), buf, indent+1)
				if err != nil {
					return err
				}
			}
			buf.WriteString("\n")
			buf.WriteString(strings.Repeat("  ", indent))
			buf.WriteString("}")
			
		case reflect.Slice, reflect.Array:
			buf.WriteString("[\n")
			indentStr := strings.Repeat("  ", indent+1)
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					buf.WriteString(",\n")
				}
				buf.WriteString(indentStr)
				err := marshalShellValue(rv.Index(i).Interface(), buf, indent+1)
				if err != nil {
					return err
				}
			}
			buf.WriteString("\n")
			buf.WriteString(strings.Repeat("  ", indent))
			buf.WriteString("]")
			
		default:
			// Fallback to string representation
			buf.WriteString(fmt.Sprintf("%v", v))
		}
	}
	
	return nil
}
