package hashing

import (
	"crypto/sha256"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Hash(result bson.M) (hash string, err error) {
	var identifier, identifiedBy string
	unknown, ok := result["_id"]
	if ok {
		identifiedBy = "_id"
	} else {
		unknown, ok = result["id"]
		if ok {
			identifiedBy = "id"
		} else {
			// todo: as future feature it should not be a problem
			err = fmt.Errorf("(not_supported_yet) no identifier field is found. Object must contain _id or id field")
			return
		}
	}

	switch idTyped := unknown.(type) {
	case string:
		identifier = idTyped
	case primitive.ObjectID:
		identifier = idTyped.Hex()
	}

	canonicalExtJson, err := bson.MarshalExtJSON(result, true, false)
	if err != nil {
		return "", fmt.Errorf("invalid bson result: %w", err)
	}

	checksum, err := CalculateChecksum(canonicalExtJson, sha256.New())
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return identifiedBy + ":" + identifier + "|" + checksum, nil
}
