package app

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

var (
	// ErrRun indicates a run-time failure.
	ErrRun = errors.New("run")

	// ErrStop indicates a stop failure.
	ErrStop = errors.New("stop")
)

// SignalError is returned when execution is interrupted by an OS signal.
type SignalError struct {
	Signal os.Signal
}

func (e *SignalError) Error() string {
	return fmt.Sprintf("signal %v received", e.Signal)
}

func (e *SignalError) ExitCode() int {
	if n, ok := e.Signal.(syscall.Signal); ok {
		return 128 + int(n)
	}

	return 1
}

// NewErrRun wraps a run-time error.
func NewErrRun(err error) error {
	return fmt.Errorf("%w: %w", ErrRun, err)
}

// NewErrStop wraps a stop error.
func NewErrStop(err error) error {
	return fmt.Errorf("%w: %w", ErrStop, err)
}

// NewErrSignal creates a new SignalError.
func NewErrSignal(sig os.Signal) error {
	return &SignalError{Signal: sig}
}
