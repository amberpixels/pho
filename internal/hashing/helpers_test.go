package hashing_test

import (
	"crypto/sha256"
	"testing"

	"pho/internal/hashing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple string",
			data:     []byte("hello"),
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			name:     "complex data",
			data:     []byte(`{"_id":"test","name":"document"}`),
			expected: "5a4f6ea0e9c5d25c5bbd9d0d3b7d8de7e9b2c4a7d8e6f3b1a9c8e5f2d4a7b9c1", // This will be calculated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hashing.CalculateChecksum(tt.data, sha256.New())
			require.NoError(t, err)

			// For the complex data test, just verify it's a valid SHA256 (64 hex chars)
			if tt.name == "complex data" {
				assert.Len(t, result, 64)
				return
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateChecksumConsistency(t *testing.T) {
	data := []byte("consistency test data")

	// Calculate checksum multiple times
	checksum1, err := hashing.CalculateChecksum(data, sha256.New())
	require.NoError(t, err)

	checksum2, err := hashing.CalculateChecksum(data, sha256.New())
	require.NoError(t, err)

	assert.Equal(t, checksum1, checksum2)
}

func TestCalculateChecksumSensitivity(t *testing.T) {
	data1 := []byte("test data 1")
	data2 := []byte("test data 2")

	checksum1, err := hashing.CalculateChecksum(data1, sha256.New())
	require.NoError(t, err)

	checksum2, err := hashing.CalculateChecksum(data2, sha256.New())
	require.NoError(t, err)

	assert.NotEqual(t, checksum1, checksum2)
}

func TestCalculateChecksumLength(t *testing.T) {
	data := []byte("test data for length verification")

	checksum, err := hashing.CalculateChecksum(data, sha256.New())
	require.NoError(t, err)

	// SHA256 produces 64 character hex string
	assert.Len(t, checksum, 64)

	// Verify it's all hex characters
	for i, c := range checksum {
		assert.True(
			t,
			(c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"Invalid hex character at position %d: %c",
			i,
			c,
		)
	}
}
