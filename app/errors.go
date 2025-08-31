package app

import (
	"fmt"
	"os"
)

// RunError wraps errors that occur during start or runtime operations.
type RunError struct {
	Err error
}

func (e *RunError) Error() string { return fmt.Sprintf("run error: %v", e.Err) }
func (e *RunError) Unwrap() error { return e.Err }

func NewRunError(err error) *RunError {
	if err == nil {
		return nil
	}

	return &RunError{Err: err}
}

// ShutdownError wraps errors that occur during shutdown operations.
type ShutdownError struct {
	Err error
}

func (e *ShutdownError) Error() string { return fmt.Sprintf("shutdown error: %v", e.Err) }
func (e *ShutdownError) Unwrap() error { return e.Err }

func NewShutdownError(err error) *ShutdownError {
	if err == nil {
		return nil
	}

	return &ShutdownError{Err: err}
}

// SignalError is returned when execution is interrupted
// by an incoming OS signal (SIGINT or SIGTERM).
type SignalError struct {
	Signal os.Signal
}

func (e *SignalError) Error() string { return fmt.Sprintf("signal %v received", e.Signal) }
