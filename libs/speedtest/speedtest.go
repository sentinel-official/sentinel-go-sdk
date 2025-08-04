package speedtest

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	"github.com/showwin/speedtest-go/speedtest"
)

// performTests runs the ping, download, and upload tests on the target server.
func performTests(ctx context.Context, target *speedtest.Server) error {
	// Perform the ping test
	if err := target.PingTestContext(ctx, nil); err != nil {
		return fmt.Errorf("failed to perform ping test: %w", err)
	}

	// Perform the download test
	if err := target.DownloadTestContext(ctx); err != nil {
		return fmt.Errorf("failed to perform download test: %w", err)
	}

	// Perform the upload test
	if err := target.UploadTestContext(ctx); err != nil {
		return fmt.Errorf("failed to perform upload test: %w", err)
	}

	// Wait for the context to be ready after the tests
	target.Context.Wait()
	return nil
}

// Run performs a speed test and returns download and upload speeds.
func Run(ctx context.Context) (dlSpeed, ulSpeed math.Int, err error) {
	// Create a new Speedtest client
	st := speedtest.New()

	// Fetch the list of servers from the Speedtest service
	servers, err := st.FetchServerListContext(ctx)
	if err != nil {
		return math.Int{}, math.Int{}, fmt.Errorf("failed to fetch servers: %w", err)
	}

	// Find the best server from the list
	targets, err := servers.FindServer(nil)
	if err != nil {
		return math.Int{}, math.Int{}, fmt.Errorf("failed to find targets: %w", err)
	}

	// Iterate through the list of target servers to find a valid result
	for _, target := range targets {
		// Perform the tests on the target server
		if err := performTests(ctx, target); err != nil {
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
	return math.Int{}, math.Int{}, errors.New("no server provided valid results")
}
