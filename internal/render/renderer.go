package render

import (
	"context"
	"fmt"
	"io"
	"pho/pkg/extjson"

	"go.mongodb.org/mongo-driver/bson"
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

	// Create unified marshaller using pkg/extjson
	var marshaller *extjson.Marshaller
	switch cfg.ExtJSONMode {
	case ExtJSONModes.Canonical:
		marshaller = extjson.NewCanonicalMarshaller()
	case ExtJSONModes.Relaxed:
		marshaller = extjson.NewRelaxedMarshaller()
	case ExtJSONModes.Shell:
		marshaller = extjson.NewShellMarshaller()
	default:
		return nil, fmt.Errorf("unsupported ExtJSON mode: %s", cfg.ExtJSONMode)
	}

	// Configure marshaller based on settings
	marshaller = marshaller.WithCompact(cfg.CompactJSON)
	if !cfg.CompactJSON {
		marshaller = marshaller.WithIndent(" ") // Default single space indent
	}

	b, err := marshaller.Marshal(result)
	if err != nil {
		if cfg.IgnoreFailures {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to marshal result: %w", err)
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

			return fmt.Errorf("failed to decode line [%d]: %w", lineNumber, err)
		}

		resultBytes, err := r.FormatResult(result)
		if err != nil {
			if cfg.IgnoreFailures {
				continue
			}

			return fmt.Errorf("failed to format line [%d]: %w", lineNumber, err)
		}

		if lineNumberBytes := r.FormatLineNumber(lineNumber); lineNumberBytes != nil {
			resultBytes = append(lineNumberBytes, resultBytes...)
		}

		if _, err = out.Write(resultBytes); err != nil {
			if cfg.IgnoreFailures {
				continue
			}

			return fmt.Errorf("failed to write line [%d]: %w", lineNumber, err)
		}

		lineNumber++
	}

	return nil
}
