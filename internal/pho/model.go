package pho

import (
	"encoding/json"
	"pho/internal/hashing"

	"go.mongodb.org/mongo-driver/bson"
)

// ParsedMeta stores hashed lines and other meta
type ParsedMeta struct {
	// Connection details for review/apply operations
	URI        string
	Database   string
	Collection string

	// Lines are hashes per identifier.
	// Identifier here is considered to be identified_by field + identifier value
	// etc. _id::111111
	Lines map[string]*hashing.HashData
}

type DumpDoc bson.M

// UnmarshalJSON for now is a hack, as we hardcode the way unmarshal parameters in here
// Whole thing of DumpDoc is required, so it's properly parsed back from ExtJson into bson
func (tx *DumpDoc) UnmarshalJSON(raw []byte) error {
	return bson.UnmarshalExtJSON(raw, true, tx)
}

// ToJSON serializes the metadata to JSON format
func (meta *ParsedMeta) ToJSON() ([]byte, error) {
	return json.MarshalIndent(meta, "", "  ")
}

// FromJSON deserializes metadata from JSON format
func (meta *ParsedMeta) FromJSON(data []byte) error {
	return json.Unmarshal(data, meta)
}
