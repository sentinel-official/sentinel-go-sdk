package core

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
	core "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
)

// Block queries a block at a given height from the Tendermint node.
// It retries the query in case of failures based on the Client's retry configuration.
// Returns the block ID, block data, or an error.
func (c *Client) Block(ctx context.Context, height int64) (*types.BlockID, *types.Block, error) {
	var result *core.ResultBlock

	var heightP *int64 = nil
	if height != 0 {
		heightP = &height
	}

	// Define the function to perform the block query.
	retryFunc := func() error {
		// Get the RPC client for querying.
		http, err := c.HTTP()
		if err != nil {
			return fmt.Errorf("creating HTTP client: %w", err)
		}

		// Perform the block query at the specified height.
		result, err = http.Block(ctx, heightP)
		if err != nil {
			return fmt.Errorf("performing block query: %w", err)
		}

		return nil
	}

	// Track the number of retry attempts.
	attempts := 0
	onRetryFunc := func(_ uint, _ error) {
		attempts++
	}

	// retryIfFunc determines whether a retry should occur based on the error.
	retryIfFunc := func(err error) bool {
		return true
	}

	// Retry the query using the configured maximum retries and delay.
	if err := retry.Do(
		retryFunc,
		retry.Context(ctx),
		retry.Attempts(c.queryRetryAttempts),
		retry.Delay(c.queryRetryDelay),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(onRetryFunc),
		retry.RetryIf(retryIfFunc),
	); err != nil {
		return nil, nil, fmt.Errorf("block query failed after %d attempt(s): %w", attempts, err)
	}

	// Return nil if no result was produced.
	if result == nil {
		return nil, nil, nil
	}

	return &result.BlockID, result.Block, nil
}
