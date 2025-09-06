package cron

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"

	"github.com/sentinel-official/sentinel-go-sdk/libs/log"
	"github.com/sentinel-official/sentinel-go-sdk/process"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Scheduler manages the scheduling and execution of workers.
type Scheduler struct {
	*process.Manager                   // Embedded process manager for handling scheduler lifecycle.
	workers          map[string]Worker // Holds all workers registered with the scheduler.
}

// NewScheduler creates and initializes a new Scheduler instance.
func NewScheduler(ctx context.Context, name string) *Scheduler {
	return &Scheduler{
		Manager: process.NewManager(ctx, name),
		workers: make(map[string]Worker),
	}
}

// Setup prepares the scheduler for operation.
func (s *Scheduler) Setup() error {
	return s.Manager.Setup(nil)
}

// Start begins executing all registered workers concurrently.
func (s *Scheduler) Start() error {
	return s.Manager.Start(func(_ context.Context) error {
		for _, val := range s.workers {
			worker := val

			s.Go(func(ctx context.Context) error {
				// Run the worker and log error if any
				if err := s.runWorker(ctx, worker); err != nil {
					if !utils.ErrorIs(context.Cause(ctx), context.Canceled) {
						log.Error("Worker exited", "cause", err, "name", worker.Name())
					}
				}

				return nil
			})
		}

		return nil
	})
}

// Wait blocks until all workers have exited or the manager is stopped.
func (s *Scheduler) Wait() error {
	return s.Manager.Wait(nil)
}

// Stop gracefully halts the scheduler and cancels all running workers.
func (s *Scheduler) Stop() error {
	return s.Manager.Stop(nil)
}

// Cleanup releases resources and finalizes the scheduler’s state after stopping.
func (s *Scheduler) Cleanup() error {
	return s.Manager.Cleanup(nil)
}

// Register adds multiple workers to the scheduler.
func (s *Scheduler) Register(workers ...Worker) error {
	if s.IsRunning() {
		return errors.New("scheduler is already running")
	}

	for _, w := range workers {
		if _, ok := s.workers[w.Name()]; ok {
			return fmt.Errorf("worker %q already exists", w.Name())
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
