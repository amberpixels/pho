package render

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"io"
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
		// TODO: implement MongoDB Ext Json v1 Shell mode
		marshal = func(v any) ([]byte, error) {
			return nil, fmt.Errorf("not implemented")
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
