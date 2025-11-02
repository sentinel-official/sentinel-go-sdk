package core

import (
	"context"
	"fmt"

	"github.com/avast/retry-go/v4"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	core "github.com/cometbft/cometbft/rpc/core/types"
)

// BlockResults queries the results of a block at a given height from the Tendermint node.
// It includes information such as transaction results, events, validator updates, and consensus parameter changes.
// Retries the query on failure based on the Client's retry configuration.
// Returns the results or an error.
func (c *Client) BlockResults(ctx context.Context, height int64) (
	[]*abci.ResponseDeliverTx, []abci.Event, []abci.Event, []abci.ValidatorUpdate, *types.ConsensusParams, error,
) {
	var result *core.ResultBlockResults

	var heightP *int64 = nil
	if height != 0 {
		heightP = &height
	}

	// Define the function to perform the block results query.
	retryFunc := func() error {
		// Get the RPC client for querying.
		http, err := c.HTTP()
		if err != nil {
			return fmt.Errorf("creating HTTP client: %w", err)
		}

		// Perform the query for block results at the specified height.
		result, err = http.BlockResults(ctx, heightP)
		if err != nil {
			return fmt.Errorf("performing block results query: %w", err)
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
		return nil, nil, nil, nil, nil, fmt.Errorf("block results query failed after %d attempt(s): %w", attempts, err)
	}

	// Return nil if no result was produced.
	if result == nil {
		return nil, nil, nil, nil, nil, nil
	}

	// Return all parts of the block results.
	return result.TxsResults, result.BeginBlockEvents, result.EndBlockEvents, result.ValidatorUpdates, result.ConsensusParamUpdates, nil
}
