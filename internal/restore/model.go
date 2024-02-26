package restore

import "fmt"

var (
	// ErrNoop is error meaning that NoopAction was trying to be restored
	// No real restore action is needed for noop.
	// So only call restoring on effective changes.
	ErrNoop = fmt.Errorf("noop")
)
