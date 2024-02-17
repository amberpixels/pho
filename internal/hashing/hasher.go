package hashing

import (
	"crypto/sha256"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"pho/pkg/extjson"
	"strings"
)

// Hash performs hashing of the given db object
// It identifies it (by _id or id field) and calculates checksum for whole its content via SHA256
// Each db object is represented via hash line: _id::123|checksum
func Hash(result bson.M) (*HashData, error) {
	var identifierValue, identifiedBy string
	unknown, ok := result["_id"]
	if ok {
		identifiedBy = "_id"
	} else {
		unknown, ok = result["id"]
		if ok {
			identifiedBy = "id"
		} else {
			// todo: as future feature it should not be a problem
			return nil, fmt.Errorf("(not_supported_yet) no identifierValue field is found. Object must contain _id or id field")
		}
	}

	switch idTyped := unknown.(type) {
	case string:
		identifierValue = idTyped
	case primitive.ObjectID:
		identifierValue = idTyped.Hex()
	}

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

type HashData struct {
	identifiedBy    string
	identifierValue string
	checksum        string
}

func (h *HashData) GetIdentifierParts() (string, string) {
	return h.identifiedBy, h.identifierValue
}

func (h *HashData) GetIdentifier() string {
	return h.identifiedBy + "::" + h.identifierValue
}

func (h *HashData) GetChecksum() string {
	return h.checksum
}

func (h *HashData) String() string {
	return h.GetIdentifier() + "|" + h.checksum
}

func Parse(hashStr string) (*HashData, error) {
	parts := strings.Split(hashStr, "|")
	if len(parts) != 2 {
		return nil, fmt.Errorf("hash string must be split by | in two parts")
	}

	subParts := strings.Split(parts[0], "::")
	if len(subParts) != 2 {
		return nil, fmt.Errorf("identifier part must be split by :: in two parts")
	}

	return &HashData{
		identifiedBy:    subParts[0],
		identifierValue: subParts[1],
		checksum:        parts[1],
	}, nil
}
