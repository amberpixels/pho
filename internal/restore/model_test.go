package restore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrNoop(t *testing.T) {
	expected := "noop"
	assert.Equal(t, expected, ErrNoop.Error())
}

func TestErrNoop_Type(t *testing.T) {
	// Test that ErrNoop is indeed an error type
	var err error = ErrNoop
	assert.NotNil(t, err)
}

func TestErrNoop_Comparison(t *testing.T) {
	// Test that we can compare with ErrNoop
	testFunc := func() error {
		return ErrNoop
	}

	err := testFunc()
	assert.Equal(t, ErrNoop, err)
}
