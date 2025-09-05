package process

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/libs/log"
	"github.com/sentinel-official/sentinel-go-sdk/process/internal"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Manager controls the lifecycle of a process, including start, stop, and cleanup.
// It ensures correct state transitions and provides concurrency-safe process management.
type Manager struct {
	name   string          // Process name.
	parent context.Context // Parent context provided at creation.
	state  internal.State  // Current state of the process.

	cancel context.CancelFunc // Cancel function for the process context.
	ctx    context.Context    // Derived context tied to this manager.
	eg     *errgroup.Group    // Errgroup to manage goroutines under this manager.

	mu sync.Mutex // Protects state and context fields from concurrent access
}

// NewManager initializes a new Manager with a given parent context, and name.
func NewManager(ctx context.Context, name string) *Manager {
	if ctx == nil {
		ctx = context.Background()
	}

	m := &Manager{
		name:   name,
		parent: ctx,
	}

	m.state.Store(internal.StateCodeUnspecified)
	return m
}

// Name returns the name of the manager.
func (m *Manager) Name() string { return m.name }

// IsRunning checks whether the manager is in a started state.
func (m *Manager) IsRunning() bool { return m.state.Load() == internal.StateCodeStarted }

// Setup runs the provided setup function with the parent context.
// It does not change the state and must only be called in the "unspecified" state.
//
// CONTRACT:
//   - MUST be called only before Start().
//   - MUST NOT spawn goroutines.
func (m *Manager) Setup(fn func(ctx context.Context) error) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state.Load() != internal.StateCodeUnspecified {
		return NewErrInvalidState(m.name)
	}

	log.Info("Setting up process", "name", m.name)

	if fn != nil {
		return fn(m.parent)
	}

	return nil
}

// Go starts a goroutine tied to the lifecycle of the manager.
//
// CONTRACT:
//   - MUST only be called synchronously inside the function passed to Start().
//   - MUST NOT be called externally.
func (m *Manager) Go(fn func(ctx context.Context) error) {
	m.eg.Go(func() error {
		// Check if context was already canceled.
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		default:
		}

		return fn(m.ctx)
	})
}

// Start transitions the manager from "unspecified" to "started" state
// and initializes a new context + errgroup for managing goroutines.
//
// CONTRACT:
//   - The fn must not block; long-running work must be placed in Go().
//   - The fn must not return an error if Go() is called inside it.
func (m *Manager) Start(fn func(ctx context.Context) error) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.state.CompareAndSwap(internal.StateCodeUnspecified, internal.StateCodeStarted) {
		return NewErrInvalidState(m.name)
	}

	defer func() {
		// If Start fails, revert state back to unspecified.
		if err != nil {
			_ = m.state.CompareAndSwap(internal.StateCodeStarted, internal.StateCodeUnspecified)
		}
	}()

	log.Info("Starting process", "name", m.name)

	// Create a fresh context tied to this manager, with cancellation support.
	ctx, cancel := context.WithCancel(m.parent)
	eg, ctx := errgroup.WithContext(ctx)

	m.cancel = cancel
	m.ctx = ctx
	m.eg = eg

	// Check if context was already canceled.
	select {
	case <-m.ctx.Done():
		return m.ctx.Err()
	default:
	}

	// Run provided start function if given.
	if fn != nil {
		return fn(m.ctx)
	}

	return nil
}

// Stop cancels the process context and marks the manager as stopped.
func (m *Manager) Stop(fn func() error) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If already stopped or not started, return nil.
	if !m.state.CompareAndSwap(internal.StateCodeStarted, internal.StateCodeStopped) {
		return nil
	}

	log.Info("Stopping process", "name", m.name)
	if m.cancel != nil {
		m.cancel()
	}

	// Run provided stop function if given.
	if fn != nil {
		return fn()
	}

	return nil
}

// Wait blocks until all goroutines started with Go() have finished.
// If an error occurs, it propagates unless the manager was stopped and context canceled.
//
// CONTRACT:
//   - MUST be called only after Start() succeeds.
func (m *Manager) Wait(fn func() error) (err error) {
	if m.state.Load() == internal.StateCodeUnspecified {
		return NewErrInvalidState(m.name)
	}

	if m.eg != nil {
		if err := m.eg.Wait(); err != nil {
			// If not stopped, propagate error.
			if m.state.Load() != internal.StateCodeStopped {
				return err
			}

			// If stopped, ignore context.Canceled errors.
			if !utils.ErrorIs(context.Cause(m.ctx), context.Canceled) {
				return err
			}
		}
	}

	// Run provided wait function if given.
	if fn != nil {
		return fn()
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

	if m.state.Load() != internal.StateCodeStopped {
		return NewErrNotStopped(m.name)
	}

	log.Info("Cleaning up process", "name", m.name)

	m.eg = nil
	m.ctx = nil
	m.cancel = nil

	// Run provided cleanup function if given.
	if fn != nil {
		if err := fn(); err != nil {
			return err
		}
	}

	return m.Reset()
}

// Reset moves the state from "stopped" back to "unspecified".
func (m *Manager) Reset() error {
	if !m.state.CompareAndSwap(internal.StateCodeStopped, internal.StateCodeUnspecified) {
		return NewErrNotStopped(m.name)
	}

	return nil
}
