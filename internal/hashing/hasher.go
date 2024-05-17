package hashing

import (
	"crypto/sha256"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"pho/pkg/extjson"
	"strings"
)

const (
	Separator = " "
)

type HashData struct {
	// identifiedBy stores the field, which data is identified by
	identifiedBy string

	// identifierValue currently can be a string or ObjectID
	identifierValue *IdentifierValue

	// checksum of the whole doc
	checksum string
}

// Hash performs hashing of the given db object
// It identifies it (by _id or id field) and calculates checksum for whole its content via SHA256
// Each db object is represented via hash line: _id::123|checksum
func Hash(result bson.M) (*HashData, error) {
	// TODO: allow via config to rewrite it
	possibleIdFields := []string{"_id", "id"}

	var identifiedBy string
	var unknown any
	var ok bool
	for _, possibleIdField := range possibleIdFields {
		if unknown, ok = result[possibleIdField]; ok {
			identifiedBy = possibleIdField
			break
		}
	}
	if !ok {
		return nil, fmt.Errorf("no identifierValue field is found. Object must contain one of %v fields", possibleIdFields)
	}

	identifierValue := NewIdentifierValue(unknown)

	canonicalExtJson, err := extjson.NewMarshaller(true).Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("invalid bson result: %w", err)
	}

	checksum, err := CalculateChecksum(canonicalExtJson, sha256.New())
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return &HashData{
		identifiedBy:    identifiedBy,
		identifierValue: identifierValue,
		checksum:        checksum,
	}, nil
}

func (h *HashData) GetIdentifierParts() (string, any) {
	return h.identifiedBy, h.identifierValue.Value
}

func (h *HashData) GetIdentifier() string {
	return h.identifiedBy + Separator + h.identifierValue.String()
}

func (h *HashData) GetChecksum() string {
	return h.checksum
}

func (h *HashData) String() string {
	return h.GetIdentifier() + Separator + h.checksum
}

func Parse(hashStr string) (*HashData, error) {
	parts := strings.Split(hashStr, Separator)
	if len(parts) != 3 {
		return nil, fmt.Errorf("hash string must be split by | in two parts")
	}

	identifierValue, err := ParseIdentifierValue(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid identifier part: %w", err)
	}

	return &HashData{
		identifiedBy:    parts[0],
		identifierValue: identifierValue,
		checksum:        parts[2],
	}, nil
}
