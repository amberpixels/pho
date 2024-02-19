package pho

import (
	"go.mongodb.org/mongo-driver/bson"
)

type DumpDoc bson.M

// UnmarshalJSON for now is a hack, as we hardcode the way unmarshal parameters in here
// Whole thing of DumpDoc is required, so it's properly parsed back from ExtJson into bson
func (tx *DumpDoc) UnmarshalJSON(raw []byte) error {
	return bson.UnmarshalExtJSON(raw, true, tx)
}
