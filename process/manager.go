package process

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/libs/log"
	"github.com/sentinel-official/sentinel-go-sdk/process/internal"
)

// Manager controls the lifecycle of a process, including start, stop, and cleanup.
// It ensures correct state transitions and provides concurrency-safe process management.
type Manager struct {
	log    log.Logger      // Logger for structured logging.
	name   string          // Process name (used in error messages).
	parent context.Context // Parent context provided at creation.
	state  internal.State  // Current state of the process.

	cancel context.CancelFunc // Cancel function for the process context.
	ctx    context.Context    // Derived context tied to this manager.
	eg     *errgroup.Group    // Errgroup to manage goroutines under this manager.
}

// NewManager initializes a new Manager with a given parent context, and name.
func NewManager(ctx context.Context, name string) *Manager {
	if ctx == nil {
		ctx = context.Background()
	}

	m := &Manager{
		log:    log.With("name", name),
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

// Go starts a goroutine tied to the lifecycle of the manager.
// Panics if the state is not Started.
func (m *Manager) Go(fn func(ctx context.Context) error) {
	if m.state.Load() != internal.StateCodeStarted || m.eg == nil {
		panic(fmt.Errorf("cannot call Go: %w", NewErrNotStarted(m.name)))
	}

	m.eg.Go(func() error {
		select {
		case <-m.ctx.Done():
			// If context is canceled, return its error.
			return m.ctx.Err()
		default:
			// Otherwise, run the provided function.
			return fn(m.ctx)
		}
	})
}

// Start transitions the manager from "unspecified" to "started" state,
// and initializes a new context + errgroup for managing goroutines.
func (m *Manager) Start(fn func(ctx context.Context) error) (err error) {
	if !m.state.CompareAndSwap(internal.StateCodeUnspecified, internal.StateCodeStarted) {
		return NewErrInvalidState(m.name)
	}

	defer func() {
		// If Start fails, revert state back to unspecified.
		if err != nil {
			_ = m.state.CompareAndSwap(internal.StateCodeStarted, internal.StateCodeUnspecified)
		}
	}()

	m.log.Info("Starting process")

	// Create a fresh context tied to this manager, with cancellation support.
	ctx, cancel := context.WithCancel(m.parent)
	eg, ctx := errgroup.WithContext(ctx)

	m.cancel = cancel
	m.ctx = ctx
	m.eg = eg

	// Check if context was already canceled.
	if m.ctx != nil {
		select {
		case <-m.ctx.Done():
			return m.ctx.Err()
		default:
		}
	}

	// Run provided start function if given.
	if fn != nil {
		return fn(m.ctx)
	}

	return nil
}

// Stop transitions the manager to the "stopped" state and cancels the context.
// Optional cleanup logic can be provided via fn.
func (m *Manager) Stop(fn func() error) (err error) {
	// If already stopped or not started, return nil.
	if !m.state.CompareAndSwap(internal.StateCodeStarted, internal.StateCodeStopped) {
		return nil
	}

	m.log.Info("Stopping process")

	if m.cancel != nil {
		m.cancel()
	}

	if fn != nil {
		return fn()
	}

	return nil
}

// Wait blocks until all goroutines under the manager have finished.
// If an error occurs, it propagates unless the manager was stopped and context canceled.
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
			if !errors.Is(context.Cause(m.ctx), context.Canceled) {
				return err
			}
		}
	}

	// Run optional callback after wait.
	if fn != nil {
		return fn()
	}

	return nil
}

// Cleanup releases resources and prepares the manager for reuse.
//
// CONTRACT:
//   - MUST only be called after Stop() has transitioned the manager to the "stopped" state.
//   - Calling Cleanup earlier will return ErrNotStopped.
//   - MUST NOT be called concurrently with Start, Go, Wait, or Stop.
//     (because Cleanup nils eg, ctx, and cancel — racing with other
//     methods that use them would cause panics or data races).
//
// Effects:
//   - Clears eg, ctx, and cancel references.
//   - Runs the optional cleanup function fn.
//   - Transitions state from "stopped" back to "unspecified" via Reset(),
//     allowing the manager to be started again.
func (m *Manager) Cleanup(fn func() error) error {
	if m.state.Load() != internal.StateCodeStopped {
		return NewErrNotStopped(m.name)
	}

	m.eg = nil
	m.ctx = nil
	m.cancel = nil

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
