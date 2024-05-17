package pho

import (
	"go.mongodb.org/mongo-driver/bson"
	"pho/internal/hashing"
)

// ParsedMeta stores hashed lines and other meta
type ParsedMeta struct {
	dbName         string
	collectionName string

	// Lines are hashes per identifier.
	// Identifier here is considered to be identified_by field + identifier value
	// etc. _id|111111
	Lines map[string]*hashing.HashData

	// TODO:
	// ExtJSON params
}

type DumpDoc bson.M

// UnmarshalJSON for now is a hack, as we hardcode the way unmarshal parameters in here
// Whole thing of DumpDoc is required, so it's properly parsed back from ExtJson into bson
func (tx *DumpDoc) UnmarshalJSON(raw []byte) error {
	return bson.UnmarshalExtJSON(raw, true, tx)
}
