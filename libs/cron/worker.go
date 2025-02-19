package cron

import (
	"time"
)

// Worker defines the interface for a scheduler worker.
type Worker interface {
	Interval() time.Duration         // Returns the interval at which the worker should be run.
	MaxRuns() uint                   // Returns the maximum number of times the worker should run (0 for unlimited).
	Name() string                    // Returns the name of the worker.
	OnError(err error) (stop bool)   // Handles errors that occur during worker execution. Returns true to stop the worker, false otherwise.
	OnExit()                         // Called when the worker stops to perform cleanup actions.
	OnRetry(attempt uint, err error) // Called after each failed attempt, passing the attempt count and error.
	RetryAttempts() uint             // Returns the number of retry attempts for the worker.
	RetryDelay() time.Duration       // Returns the delay between retry attempts.
	Run() error                      // Executes the worker and returns an error if it fails.
}

// Ensure BasicWorker implements the Worker interface.
var _ Worker = (*BasicWorker)(nil)

// BasicWorker provides a basic implementation of the Worker interface.
type BasicWorker struct {
	handler       func() error
	maxRuns       uint
	interval      time.Duration
	name          string
	onError       func(error) bool
	onExit        func()
	onRetry       func(uint, error)
	retryAttempts uint
	retryDelay    time.Duration
}

// NewBasicWorker creates a new BasicWorker with default settings.
func NewBasicWorker() *BasicWorker {
	w := &BasicWorker{}
	w.WithOnError(func(_ error) bool { return false })
	w.WithOnExit(func() {})
	w.WithOnRetry(func(_ uint, _ error) {})
	w.WithRetryAttempts(1)
	w.WithRetryDelay(1 * time.Second)

	return w
}

// WithHandler sets the handler function for the worker.
func (w *BasicWorker) WithHandler(handler func() error) *BasicWorker {
	w.handler = handler
	return w
}

// WithInterval sets the interval at which the worker should be run.
func (w *BasicWorker) WithInterval(interval time.Duration) *BasicWorker {
	w.interval = interval
	return w
}

// WithMaxRuns sets the maximum number of times the worker should run.
func (w *BasicWorker) WithMaxRuns(runs uint) *BasicWorker {
	w.maxRuns = runs
	return w
}

// WithName sets the name of the worker.
func (w *BasicWorker) WithName(name string) *BasicWorker {
	w.name = name
	return w
}

// WithOnError sets the function to handle errors that occur during worker execution.
func (w *BasicWorker) WithOnError(onError func(error) bool) *BasicWorker {
	w.onError = onError
	return w
}

// WithOnExit sets the function to be called when the worker stops.
func (w *BasicWorker) WithOnExit(onExit func()) *BasicWorker {
	w.onExit = onExit
	return w
}

// WithOnRetry sets the function to be called after each failed attempt.
func (w *BasicWorker) WithOnRetry(onRetry func(uint, error)) *BasicWorker {
	w.onRetry = onRetry
	return w
}

// WithRetryAttempts sets the number of retry attempts for the worker.
func (w *BasicWorker) WithRetryAttempts(attempts uint) *BasicWorker {
	w.retryAttempts = attempts
	return w
}

// WithRetryDelay sets the delay between retry attempts.
func (w *BasicWorker) WithRetryDelay(delay time.Duration) *BasicWorker {
	w.retryDelay = delay
	return w
}

// Interval returns the interval at which the worker should be executed.
func (w *BasicWorker) Interval() time.Duration {
	return w.interval
}

// MaxRuns returns the maximum number of times the worker should run (0 for unlimited).
func (w *BasicWorker) MaxRuns() uint {
	return w.maxRuns
}

// Name returns the name of the worker.
func (w *BasicWorker) Name() string {
	return w.name
}

// OnError processes errors encountered during worker execution.
func (w *BasicWorker) OnError(err error) bool {
	if w.onError != nil {
		return w.onError(err)
	}

	return false
}

// OnExit calls the onExit function if it is set.
func (w *BasicWorker) OnExit() {
	if w.onExit != nil {
		w.onExit()
	}
}

// OnRetry processes retry attempts for the worker.
func (w *BasicWorker) OnRetry(attempt uint, err error) {
	if w.onRetry != nil {
		w.onRetry(attempt, err)
	}
}

// RetryAttempts returns the number of retry attempts for the worker.
func (w *BasicWorker) RetryAttempts() uint {
	return w.retryAttempts
}

// RetryDelay returns the delay between retry attempts.
func (w *BasicWorker) RetryDelay() time.Duration {
	return w.retryDelay
}

// Run executes the worker's handler function and returns any error encountered.
func (w *BasicWorker) Run() error {
	if w.handler != nil {
		return w.handler()
	}

	return nil
}
