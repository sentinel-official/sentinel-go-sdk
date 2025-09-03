package process

import (
	"errors"
	"fmt"
)

// Predefined error values for different invalid states.
var (
	ErrInvalidState = errors.New("invalid state")
	ErrNotStarted   = errors.New("not started")
	ErrNotStopped   = errors.New("not stopped")
)

// NewErrInvalidState formats an error indicating invalid state for a named process.
func NewErrInvalidState(name string) error {
	return fmt.Errorf("process %s: %w", name, ErrInvalidState)
}

// NewErrNotStarted formats an error indicating the process has not started yet.
func NewErrNotStarted(name string) error {
	return fmt.Errorf("process %s: %w", name, ErrNotStarted)
}

// NewErrNotStopped formats an error indicating the process has not stopped yet.
func NewErrNotStopped(name string) error {
	return fmt.Errorf("process %s: %w", name, ErrNotStopped)
}
