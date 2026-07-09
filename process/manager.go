package process

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/libs/log"
	"github.com/sentinel-official/sentinel-go-sdk/process/internal"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Manager controls the lifecycle of a process, including start, stop, and cleanup.
// It ensures correct state transitions and provides concurrency-safe process management.
type Manager struct {
	name  string         // Process name.
	state internal.State // Current state of the process.

	cancel context.CancelFunc // Cancel function for the process context.
	eg     *errgroup.Group    // Errgroup to manage goroutines under this manager.

	mu sync.Mutex // Protects state and context fields from concurrent access
}

// NewManager initializes a new Manager with a given parent context, and name.
func NewManager(name string) *Manager {
	return &Manager{
		name: name,
	}
}

// Name returns the name of the manager.
func (m *Manager) Name() string { return m.name }

// IsRunning checks whether the manager is in starting or started state.
func (m *Manager) IsRunning() bool {
	return m.state.Is(internal.StateStarting, internal.StateStarted)
}

// Go starts a goroutine tied to the lifecycle of the manager.
//
// CONTRACT:
//   - MUST only be called synchronously inside the function passed to Start().
//   - MUST NOT be called externally.
func (m *Manager) Go(ctx context.Context, fn func() error) {
	m.eg.Go(func() error {
		// Check if context was already canceled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Run provided go function if given.
		if fn != nil {
			if err := fn(); err != nil {
				return fmt.Errorf("calling: %w", err)
			}
		}

		return nil
	})
}

// Setup runs the provided setup function with the parent context.
// It does not change the state and must only be called in the "unspecified" state.
//
// CONTRACT:
//   - MUST be called only before Start().
//   - MUST NOT spawn goroutines.
func (m *Manager) Setup(ctx context.Context, fn func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.state.Is(internal.StateUnspecified) {
		return NewErrInvalidState(m.name)
	}

	log.Debug("Setting up process", "name", m.name)

	// Check if context was already canceled.
	select {
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck
	default:
	}

	// Run provided setup function if given.
	if fn != nil {
		if err := fn(); err != nil {
			return fmt.Errorf("calling: %w", err)
		}
	}

	return nil
}

// Start transitions the manager from "unspecified" to "starting" state,
// and then to either "started" or "start error" based on success or failure.
// It also initializes a new context and errgroup for managing goroutines.
//
// CONTRACT:
//   - The fn must not block; long-running work must be placed in Go().
//   - The fn must not return an error if Go() is called inside it.
func (m *Manager) Start(parent context.Context, fn func(ctx context.Context) error) (_ context.Context, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.state.SetIf(internal.StateStarting, internal.StateUnspecified); !ok {
		return nil, NewErrInvalidState(m.name)
	}

	defer func() {
		state := internal.StateStarted
		if err != nil {
			state = internal.StateStartError
		}

		if _, ok := m.state.SetIf(state, internal.StateStarting); !ok {
			panic(NewErrInvalidState(m.name))
		}
	}()

	log.Debug("Starting process", "name", m.name)

	// Create a fresh context tied to this manager, with cancellation support.
	ctx, cancel := context.WithCancel(parent)
	eg, ctx := errgroup.WithContext(ctx)

	m.cancel = cancel
	m.eg = eg

	// Check if context was already canceled.
	select {
	case <-ctx.Done():
		return nil, ctx.Err() //nolint:wrapcheck
	default:
	}

	// Run provided start function if given.
	if fn != nil {
		if err := fn(ctx); err != nil {
			return nil, fmt.Errorf("calling: %w", err)
		}
	}

	return ctx, nil
}

// Stop cancels the process context and transitions the manager's state
// to "stopping", then to either "stopped" or "stop error" based on success or failure.
func (m *Manager) Stop(fn func() error) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If already stopped or not started, return nil.
	if _, ok := m.state.SetIfNot(internal.StateStopping, internal.StateUnspecified); !ok {
		return nil
	}

	defer func() {
		state := internal.StateStopped
		if err != nil {
			state = internal.StateStopError
		}

		if _, ok := m.state.SetIf(state, internal.StateStopping); !ok {
			panic(NewErrInvalidState(m.name))
		}
	}()

	log.Debug("Stopping process", "name", m.name)

	if m.cancel != nil {
		m.cancel()
	}

	// Run provided stop function if given.
	if fn != nil {
		if err := fn(); err != nil {
			return fmt.Errorf("calling: %w", err)
		}
	}

	return nil
}

// Wait blocks until all goroutines started with Go() have finished.
// If an error occurs, it propagates unless the manager was stopped and context canceled.
//
// CONTRACT:
//   - MUST be called only after Start() succeeds.
func (m *Manager) Wait(ctx context.Context, fn func() error) error {
	if m.state.Is(internal.StateUnspecified) {
		return NewErrInvalidState(m.name)
	}

	if m.eg != nil {
		if err := m.eg.Wait(); err != nil {
			// If not stopped, propagate error.
			if !m.state.Is(internal.StateStopping, internal.StateStopError, internal.StateStopped) {
				return fmt.Errorf("waiting process group: %w", err)
			}

			// If stopped, ignore context.Canceled errors.
			if !utils.ErrorIs(context.Cause(ctx), context.Canceled) {
				return fmt.Errorf("waiting process group: %w", err)
			}
		}
	}

	// Run provided wait function if given.
	if fn != nil {
		if err := fn(); err != nil {
			return fmt.Errorf("calling: %w", err)
		}
	}

	return nil
}

// Cleanup releases resources and prepares the manager for reuse.
//
// CONTRACT:
//   - MUST only be called after Stop() and Wait() have both completed.
//   - MUST NOT be called concurrently with any other lifecycle method.
func (m *Manager) Cleanup(fn func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.state.Is(internal.StateStopped) {
		return NewErrNotStopped(m.name)
	}

	log.Debug("Cleaning up process", "name", m.name)

	m.eg = nil
	m.cancel = nil

	// Run provided cleanup function if given.
	if fn != nil {
		if err := fn(); err != nil {
			return fmt.Errorf("calling: %w", err)
		}
	}

	return m.Reset()
}

// Reset moves the state from "stopped" back to "unspecified".
func (m *Manager) Reset() error {
	if _, ok := m.state.SetIf(internal.StateUnspecified, internal.StateStopped); !ok {
		return NewErrNotStopped(m.name)
	}

	return nil
}
