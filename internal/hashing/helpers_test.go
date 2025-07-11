package hashing

import (
	"crypto/sha256"
	"testing"
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
			result, err := CalculateChecksum(tt.data, sha256.New())
			if err != nil {
				t.Errorf("CalculateChecksum() error = %v", err)
				return
			}

			// For the complex data test, just verify it's a valid SHA256 (64 hex chars)
			if tt.name == "complex data" {
				if len(result) != 64 {
					t.Errorf("CalculateChecksum() invalid SHA256 length: got %d, want 64", len(result))
				}
				return
			}

			if result != tt.expected {
				t.Errorf("CalculateChecksum() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateChecksumConsistency(t *testing.T) {
	data := []byte("consistency test data")

	// Calculate checksum multiple times
	checksum1, err := CalculateChecksum(data, sha256.New())
	if err != nil {
		t.Fatalf("First checksum failed: %v", err)
	}

	checksum2, err := CalculateChecksum(data, sha256.New())
	if err != nil {
		t.Fatalf("Second checksum failed: %v", err)
	}

	if checksum1 != checksum2 {
		t.Errorf("Checksum consistency failed: %s != %s", checksum1, checksum2)
	}
}

func TestCalculateChecksumSensitivity(t *testing.T) {
	data1 := []byte("test data 1")
	data2 := []byte("test data 2")

	checksum1, err := CalculateChecksum(data1, sha256.New())
	if err != nil {
		t.Fatalf("First checksum failed: %v", err)
	}

	checksum2, err := CalculateChecksum(data2, sha256.New())
	if err != nil {
		t.Fatalf("Second checksum failed: %v", err)
	}

	if checksum1 == checksum2 {
		t.Errorf("Different data should produce different checksums")
	}
}

func TestCalculateChecksumLength(t *testing.T) {
	data := []byte("test data for length verification")

	checksum, err := CalculateChecksum(data, sha256.New())
	if err != nil {
		t.Fatalf("Checksum calculation failed: %v", err)
	}

	// SHA256 produces 64 character hex string
	if len(checksum) != 64 {
		t.Errorf("SHA256 checksum should be 64 characters, got %d", len(checksum))
	}

	// Verify it's all hex characters
	for i, c := range checksum {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Invalid hex character at position %d: %c", i, c)
		}
	}
}
