package speedtest

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	"github.com/showwin/speedtest-go/speedtest"
)

// performTests runs the ping, download, and upload tests on the target server.
// The library ignores the context for test termination, so it is honored here.
func performTests(ctx context.Context, s *speedtest.Server) error {
	done := make(chan error, 1)

	go func() {
		done <- func() error {
			if err := s.PingTestContext(ctx, nil); err != nil {
				return fmt.Errorf("performing ping test on server %q: %w", s.Name, err)
			}

			if err := s.DownloadTestContext(ctx); err != nil {
				return fmt.Errorf("performing download test on server %q: %w", s.Name, err)
			}

			if err := s.UploadTestContext(ctx); err != nil {
				return fmt.Errorf("performing upload test on server %q: %w", s.Name, err)
			}

			s.Context.Wait()

			return nil
		}()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Run performs a speed test and returns download and upload speeds.
func Run(ctx context.Context) (dlSpeed, ulSpeed math.Int, err error) {
	// Create a new Speedtest client
	st := speedtest.New()

	// Fetch the list of servers from the Speedtest service
	servers, err := st.FetchServerListContext(ctx)
	if err != nil {
		return math.Int{}, math.Int{}, fmt.Errorf("fetching speedtest servers: %w", err)
	}

	// Find the best server from the list
	targets, err := servers.FindServer(nil)
	if err != nil {
		return math.Int{}, math.Int{}, fmt.Errorf("finding optimal servers: %w", err)
	}

	// Iterate through the list of target servers to find a valid result
	for _, target := range targets {
		// Stop if the context has been canceled.
		if err := ctx.Err(); err != nil {
			return math.Int{}, math.Int{}, err
		}

		// Perform the tests on the target server
		if err := performTests(ctx, target); err != nil {
			// Abort on cancellation instead of trying the next server.
			if ctx.Err() != nil {
				return math.Int{}, math.Int{}, ctx.Err()
			}

			target.Context.Reset()

			continue
		}

		// Convert download and upload speeds to math.LegacyDec
		dlSpeedDec, err := math.LegacyNewDecFromStr(fmt.Sprintf("%f", target.DLSpeed))
		if err != nil {
			target.Context.Reset()

			continue
		}

		ulSpeedDec, err := math.LegacyNewDecFromStr(fmt.Sprintf("%f", target.ULSpeed))
		if err != nil {
			target.Context.Reset()

			continue
		}

		// Convert LegacyDec to math.Int
		dlSpeed := dlSpeedDec.RoundInt()
		ulSpeed := ulSpeedDec.RoundInt()

		// Check if the speeds are positive
		if !dlSpeed.IsPositive() || !ulSpeed.IsPositive() {
			target.Context.Reset()

			continue
		}

		// A valid result was found, exit the loop
		return dlSpeed, ulSpeed, nil
	}

	// Return an error if no valid result was found
	return math.Int{}, math.Int{}, errors.New("no servers returned valid speedtest results")
}
