package internal

import (
	"sync/atomic"
)

// StateCode represents the state of a process.
// It uses int32 internally for atomic operations.
type StateCode int32

const (
	StateCodeUnspecified StateCode = iota // Default state (not started yet).
	StateCodeStarted                      // Process has started.
	StateCodeStopped                      // Process has been stopped.
)

// State wraps an atomic integer to safely manage state transitions
// between multiple goroutines.
type State struct {
	v atomic.Int32
}

// Load returns the current state atomically.
func (s *State) Load() StateCode {
	return StateCode(s.v.Load())
}

// Store sets the state atomically to the given value.
func (s *State) Store(val StateCode) {
	s.v.Store(int32(val))
}

// Swap atomically replaces the state with a new one and returns the old state.
func (s *State) Swap(new StateCode) (old StateCode) {
	return StateCode(s.v.Swap(int32(new)))
}

// CompareAndSwap atomically sets the state to 'new' if the current state matches 'old'.
// Returns true if the swap was successful.
func (s *State) CompareAndSwap(old, new StateCode) (swapped bool) {
	return s.v.CompareAndSwap(int32(old), int32(new))
}
