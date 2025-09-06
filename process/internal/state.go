package internal

import (
	"sync"
)

// StateCode represents the state of a process.
type StateCode int32

const (
	StateUnspecified StateCode = iota
	StateStarting
	StateStartError
	StateStarted
	StateStopping
	StateStopError
	StateStopped
)

// State wraps a RWMutex-protected integer to safely manage state transitions.
type State struct {
	mu   sync.RWMutex
	code StateCode
}

// Get returns the current state.
func (s *State) Get() StateCode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.code
}

// Set unconditionally sets the state to 'new'.
// Returns the previous state.
func (s *State) Set(new StateCode) StateCode {
	s.mu.Lock()
	defer s.mu.Unlock()

	old := s.code
	s.code = new

	return old
}

// SetIf sets the state to 'new' only if the current state matches any of the given 'olds' states.
// Returns the previous state and true if the state was changed, false if no change occurred.
func (s *State) SetIf(new StateCode, olds ...StateCode) (old StateCode, changed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	old = s.code

	// Check if the current state matches any of the 'olds'
	for _, code := range olds {
		if old == code {
			s.code = new
			return old, true
		}
	}

	// No change if no match
	return old, false
}

// SetIfNot sets the state to 'new' only if the current state does NOT match any of the given 'olds' states.
// Returns the previous state and true if the state was changed, false if no change occurred.
func (s *State) SetIfNot(new StateCode, olds ...StateCode) (old StateCode, changed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	old = s.code

	// Check if the current state matches any of the 'olds'
	for _, code := range olds {
		if old == code {
			// No change if there's a match
			return old, false
		}
	}

	// Set the new state if no match is found
	s.code = new
	return old, true
}

// Is reports whether the current state matches any of the given states.
func (s *State) Is(states ...StateCode) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, code := range states {
		if s.code == code {
			return true
		}
	}

	return false
}
