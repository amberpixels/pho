package hashing

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"pho/pkg/extjson"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	IdentifierSeparator = "::"
	ChecksumSeparator   = "|"
)

type HashData struct {
	// IdentifiedBy stores the field, which data is identified by
	IdentifiedBy string `json:"identified_by"`

	// IdentifierValue currently can be a string or ObjectID
	IdentifierValue *IdentifierValue `json:"identifier_value"`

	// Checksum of the whole doc
	Checksum string `json:"checksum"`
}

// Hash performs hashing of the given db object
// It identifies it (by _id or id field) and calculates checksum for whole its content via SHA256
// Each db object is represented via hash line: _id::123|checksum.
func Hash(result bson.M) (*HashData, error) {
	// TODO: allow via config to rewrite it
	possibleIDFields := []string{"_id", "id"}

	var identifiedBy string
	var unknown any
	var ok bool
	for _, possibleIDField := range possibleIDFields {
		if unknown, ok = result[possibleIDField]; ok {
			identifiedBy = possibleIDField
			break
		}
	}
	if !ok {
		return nil, fmt.Errorf(
			"no identifierValue field is found. Object must contain one of %v fields",
			possibleIDFields,
		)
	}

	identifierValue := NewIdentifierValue(unknown)

	canonicalExtJSON, err := extjson.NewCanonicalMarshaller().Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("invalid bson result: %w", err)
	}

	checksum, err := CalculateChecksum(canonicalExtJSON, sha256.New())
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return &HashData{
		IdentifiedBy:    identifiedBy,
		IdentifierValue: identifierValue,
		Checksum:        checksum,
	}, nil
}

func (h *HashData) GetIdentifierParts() (string, any) {
	if h.IdentifierValue == nil {
		return h.IdentifiedBy, nil
	}
	return h.IdentifiedBy, h.IdentifierValue.Value
}

func (h *HashData) GetIdentifier() string {
	return h.IdentifiedBy + IdentifierSeparator + h.IdentifierValue.String()
}

func (h *HashData) GetChecksum() string {
	return h.Checksum
}

func (h *HashData) String() string {
	return h.GetIdentifier() + ChecksumSeparator + h.Checksum
}

func Parse(hashStr string) (*HashData, error) {
	identifierPart, checksum, found := strings.Cut(hashStr, ChecksumSeparator)
	if !found {
		return nil, errors.New("hash string must contain checksum separator |")
	}

	identifiedBy, identifierValueStr, found := strings.Cut(identifierPart, IdentifierSeparator)
	if !found {
		return nil, errors.New("identifier part must contain identifier separator")
	}

	identifierValue, err := ParseIdentifierValue(identifierValueStr)
	if err != nil {
		return nil, fmt.Errorf("invalid identifier part: %w", err)
	}

	return &HashData{
		IdentifiedBy:    identifiedBy,
		IdentifierValue: identifierValue,
		Checksum:        checksum,
	}, nil
}
