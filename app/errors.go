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

	// ErrShutdown indicates a shutdown failure.
	ErrShutdown = errors.New("shutdown")
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

// NewErrShutdown wraps a shutdown error.
func NewErrShutdown(err error) error {
	return fmt.Errorf("%w: %w", ErrShutdown, err)
}

// NewErrSignal creates a new SignalError.
func NewErrSignal(sig os.Signal) error {
	return &SignalError{Signal: sig}
}
