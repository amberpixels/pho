package restore

import (
	"testing"
)

func TestErrNoop(t *testing.T) {
	expected := "noop"
	if ErrNoop.Error() != expected {
		t.Errorf("ErrNoop.Error() = %v, want %v", ErrNoop.Error(), expected)
	}
}

func TestErrNoop_Type(t *testing.T) {
	// Test that ErrNoop is indeed an error type
	var err error = ErrNoop
	if err == nil {
		t.Error("ErrNoop should implement error interface")
	}
}

func TestErrNoop_Comparison(t *testing.T) {
	// Test that we can compare with ErrNoop
	testFunc := func() error {
		return ErrNoop
	}

	err := testFunc()
	if err != ErrNoop {
		t.Error("Function should return ErrNoop")
	}
}
