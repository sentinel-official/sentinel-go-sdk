package cron

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/avast/retry-go/v4"

	"github.com/sentinel-official/sentinel-go-sdk/libs/log"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Scheduler manages the scheduling and execution of workers.
type Scheduler struct {
	running atomic.Bool       // Tracks whether the scheduler is currently running (atomic for thread-safety).
	workers map[string]Worker // Holds all workers registered with the scheduler.

	cancel context.CancelFunc // Cancels the shared context, stopping all workers.
	ctx    context.Context    // Shared context used by all workers for cancellation and deadlines.
	wg     *sync.WaitGroup    // WaitGroup that manages and waits for all worker goroutines.
}

// NewScheduler creates and initializes a new Scheduler instance.
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		workers: make(map[string]Worker),
		cancel:  cancel,
		ctx:     ctx,
		wg:      &sync.WaitGroup{},
	}
}

// Start begins executing all registered workers concurrently.
func (s *Scheduler) Start() error {
	// Atomically check and set running
	if s.running.Swap(true) {
		return errors.New("scheduler is already running")
	}

	for _, val := range s.workers {
		worker := val

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			// Run the worker and log error if any
			if err := s.runWorker(s.ctx, worker); err != nil {
				if !utils.ErrorIs(err, context.Canceled) {
					log.Error("Worker exited", "cause", err, "name", worker.Name())
				}
			}
		}()
	}

	return nil
}

// Wait blocks until all workers have completed (or one fails).
func (s *Scheduler) Wait() error {
	defer s.running.Store(false)

	// Wait for all worker goroutines to finish
	s.wg.Wait()

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
			return ctx.Err()
		case <-time.After(w.Interval()):
		}
	}

	return nil
}
