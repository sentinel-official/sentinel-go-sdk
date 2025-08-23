package cron

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"
	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/libs/log"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Scheduler manages the scheduling and execution of workers.
type Scheduler struct {
	running atomic.Bool       // Tracks whether the scheduler is currently running (atomic for thread-safety).
	workers map[string]Worker // Holds all workers registered with the scheduler.

	cancel context.CancelFunc // Cancels the shared context, stopping all workers.
	ctx    context.Context    // Shared context used by all workers for cancellation and deadlines.
	eg     *errgroup.Group    // Errgroup that manages and waits for all worker goroutines.
}

// NewScheduler creates and initializes a new Scheduler instance.
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Scheduler{
		workers: make(map[string]Worker),
		cancel:  cancel,
		ctx:     ctx,
		eg:      eg,
	}
}

// Start begins executing all registered workers concurrently using errgroup.
func (s *Scheduler) Start(ctx context.Context) error {
	// Atomically check and set running
	if !s.running.CompareAndSwap(false, true) {
		return errors.New("scheduler is already running")
	}

	// Combine the scheduler's context and the passed context
	ctx, _ = utils.AnyDoneContext(s.ctx, ctx)

	for _, w := range s.workers {
		worker := w
		s.eg.Go(func() error {
			// Run the worker and log errors without propagating them
			if err := s.runWorker(ctx, worker); err != nil {
				log.Error("Worker exited with error", "cause", err, "name", worker.Name())
			}

			return nil
		})
	}

	return nil
}

// Wait blocks until all workers have completed (or one fails).
func (s *Scheduler) Wait() error {
	defer s.running.Store(false)

	// Wait for all worker goroutines to finish
	if err := s.eg.Wait(); err != nil {
		return err
	}

	return nil
}

// Stop cancels the context, halting all workers.
func (s *Scheduler) Stop() error {
	// Cancel background tasks if any.
	s.cancel()

	return nil
}

// RegisterWorkers adds multiple workers to the scheduler.
func (s *Scheduler) RegisterWorkers(workers ...Worker) error {
	if s.running.Load() {
		return errors.New("scheduler is already running")
	}

	for _, w := range workers {
		if _, ok := s.workers[w.Name()]; ok {
			return fmt.Errorf("duplicate worker %s", w.Name())
		}

		s.workers[w.Name()] = w
	}

	return nil
}

// runWorker continuously executes a worker's function and handles errors.
func (s *Scheduler) runWorker(ctx context.Context, w Worker) error {
	defer w.OnExit()

	for c := uint(0); w.MaxRuns() == 0 || c < w.MaxRuns(); c++ {
		// Attempt the worker's run function with retries
		if err := retry.Do(
			func() error { return w.Run(ctx) },
			retry.Context(ctx),
			retry.Attempts(w.RetryAttempts()),
			retry.Delay(w.RetryDelay()),
			retry.DelayType(retry.FixedDelay),
			retry.OnRetry(w.OnRetry),
			retry.LastErrorOnly(true),
		); err != nil {
			if exit := w.OnError(err); exit {
				return err
			}
		}

		// Sleep for the interval—or stop early if the context is done
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(w.Interval()):
		}
	}

	return nil
}
