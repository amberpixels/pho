package hashing

import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
)

// IdentifierValue stores the X value of `{_id:X}` identifying pair
type IdentifierValue struct {
	// Value possibly now: string | primitive.ObjectID
	Value any
}

func NewIdentifierValue(v any) *IdentifierValue {
	id := &IdentifierValue{Value: v}

	// TODO: reconsider as this smells too much
	// String() contains validation as well, so let's panic as earlier as possible
	_ = id.String()

	return id
}

// String returns string representation used in meta and output
func (id *IdentifierValue) String() string {
	switch t := id.Value.(type) {
	case string:
		return t
	case fmt.Stringer: // primitive.ObjectID is a stringer, but any stringer is fine
		return t.String()

	default:
		panic(fmt.Sprintf("invalid identifier data type: %T", t))
	}
}

// ParseIdentifierValue here does the reverse operation of String()
// e.g. string `ObjectID("X")` will become an actual primitive.ObjectID
func ParseIdentifierValue(s string) (*IdentifierValue, error) {
	// TODO: rewrite via regex
	if strings.HasPrefix(s, `ObjectID("`) && strings.HasSuffix(s, `")`) {
		hex, found := strings.CutPrefix(s, `ObjectID("`)
		if found {
			hex, _ = strings.CutSuffix(hex, `")`)
		}

		oid, err := primitive.ObjectIDFromHex(hex)
		if err != nil {
			return nil, fmt.Errorf("invalid hex: %w", err)
		}

		return &IdentifierValue{Value: oid}, nil
	}

	// TODO: allow via flag plaintext string ids
	// For now: any other string is considered to be simply a hex string
	oid, err := primitive.ObjectIDFromHex(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	return &IdentifierValue{Value: oid.Hex()}, nil
}
