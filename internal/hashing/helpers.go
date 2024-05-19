package hashing

import (
	"encoding/hex"
	"fmt"
	"hash"
)

// CalculateChecksum calculates the checksum for the given data via given hash algorithm
func CalculateChecksum(data []byte, hash hash.Hash) (string, error) {
	// Write the data to the hash object
	_, err := hash.Write(data)
	if err != nil {
		return "", fmt.Errorf("hash error: %w", err)
	}

	// Get the computed hash sum
	checksum := hash.Sum(nil)

	// Convert the checksum to a hexadecimal string
	checksumHex := hex.EncodeToString(checksum)

	return checksumHex, nil
}
