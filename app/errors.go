package app

import (
	"fmt"
	"os"
)

// StartError wraps errors that occur during Start() operations.
type StartError struct {
	Err error
}

func (e *StartError) Error() string { return fmt.Sprintf("starting: %v", e.Err) }
func (e *StartError) Unwrap() error { return e.Err }

// StopError wraps errors that occur during Stop() operations.
type StopError struct {
	Err error
}

func (e *StopError) Error() string { return fmt.Sprintf("stopping: %v", e.Err) }
func (e *StopError) Unwrap() error { return e.Err }

// WaitError wraps errors that occur during Wait() operations.
type WaitError struct {
	Err error
}

func (e *WaitError) Error() string { return fmt.Sprintf("waiting: %v", e.Err) }
func (e *WaitError) Unwrap() error { return e.Err }

// SignalError is returned when execution is interrupted
// by an incoming OS signal (SIGINT or SIGTERM).
type SignalError struct {
	Signal os.Signal
}

func (e *SignalError) Error() string { return fmt.Sprintf("signal %v received", e.Signal) }
