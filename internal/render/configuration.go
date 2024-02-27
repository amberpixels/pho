package render

import "encoding/json"

// ExtJSONMode represents the Extended JSON (v2) Mode (canonical, relaxed):
// https://www.mongodb.com/docs/manual/reference/mongodb-extended-json/
// Or "shell" mode of Extended JSON (v1):
// https://www.mongodb.com/docs/manual/reference/mongodb-extended-json-v1/
type ExtJSONMode string

// ExtJSONModes is a dictionary mapping mode names to their corresponding values.
var ExtJSONModes = struct {
	Canonical ExtJSONMode
	Relaxed   ExtJSONMode
	Shell     ExtJSONMode
}{
	Canonical: "canonical",
	Relaxed:   "relaxed",
	Shell:     "shell",
}

type Configuration struct {
	// ShowLineNumbers turns on lines number via `/* 1 */` comments
	// Note: this make JSON document not valid
	ShowLineNumbers bool

	// AsValidJSON forces output to be a valid JSON
	// That means it will be returned as:
	// [
	//   {},
	//   {}
	// ]
	// Note: can't be used with ExtJSONMode="shell"
	AsValidJSON bool

	// ExtJSONMode used for current rendering
	ExtJSONMode ExtJSONMode

	// CompactJSON stands for compact output: one document per line
	CompactJSON bool

	// MinimizedJSON stands for fully minimized view: whole result is one line
	// Forces `AsValidJSON` to be true
	MinimizedJSON bool

	// IgnoreFailures stands for ignoring rendering of failed items
	// So instead of failed rendering it will return partial rendered data
	IgnoreFailures bool
}

// Option is a functional option for filling in Configuration for the Renderer.
type Option func(*Configuration)

// NewConfiguration creates a new Configuration object with the given options.
func NewConfiguration(options ...Option) *Configuration {
	config := &Configuration{}
	for _, opt := range options {
		opt(config)
	}
	return config
}

func (c *Configuration) Clone() *Configuration {
	var clone Configuration
	_ = RoundTripJSON(c, &clone)
	return &clone
}

// WithShowLineNumbers sets the ShowLineNumbers option.
func WithShowLineNumbers(v bool) Option { return func(c *Configuration) { c.ShowLineNumbers = v } }

// WithAsValidJSON sets the AsValidJSON option.
func WithAsValidJSON(v bool) Option { return func(c *Configuration) { c.AsValidJSON = v } }

// WithExtJSONMode sets the ExtJSONMode option.
func WithExtJSONMode(v ExtJSONMode) Option { return func(c *Configuration) { c.ExtJSONMode = v } }

// WithCompactJSON sets the CompactJSON option.
func WithCompactJSON(v bool) Option { return func(c *Configuration) { c.CompactJSON = v } }

// WithMinimizedJSON sets the MinimizedJSON option.
func WithMinimizedJSON(v bool) Option { return func(c *Configuration) { c.MinimizedJSON = v } }

// WithIgnoreFailures sets the IgnoreFailures option.
func WithIgnoreFailures(v bool) Option { return func(c *Configuration) { c.IgnoreFailures = v } }

// RoundTripJSON makes marshal and then unmarshal back into other given variable
// Can be used for copying things that can be JSONified
// TODO: move to some helpers utils
func RoundTripJSON(in, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, out); err != nil {
		return err
	}

	return nil
}
