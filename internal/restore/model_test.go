package restore_test

import (
	"testing"

	"pho/internal/restore"

	"github.com/stretchr/testify/assert"
)

func TestErrNoop(t *testing.T) {
	expected := "noop"
	assert.Equal(t, expected, restore.ErrNoop.Error())
}

func TestErrNoop_Type(t *testing.T) {
	// Test that restore.ErrNoop is indeed an error type
	var err = restore.ErrNoop
	assert.Error(t, err)
}

func TestErrNoop_Comparison(t *testing.T) {
	// Test that we can compare with restore.ErrNoop
	testFunc := func() error {
		return restore.ErrNoop
	}

	err := testFunc()
	assert.Equal(t, restore.ErrNoop, err)
}
