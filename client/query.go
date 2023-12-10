package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	abcitypes "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/bytes"

	"github.com/sentinel-official/sentinel-sdk/v1/client/http"
	"github.com/sentinel-official/sentinel-sdk/v1/client/options"
)

// ABCIQueryWithOptions performs an ABCI query with specified options.
func (c *Context) ABCIQueryWithOptions(ctx context.Context, path string, data bytes.HexBytes, opts *options.QueryOptions) (*abcitypes.ResponseQuery, error) {
	// Create a new HTTP client with the provided options
	client, err := http.NewFromQueryOptions(opts)
	if err != nil {
		return nil, err
	}

	// Retry the query for a maximum number of times specified by MaxRetries option
	for t := 0; t < opts.MaxRetries; t++ {
		// Perform the ABCI query using the HTTP client
		result, err := client.ABCIQueryWithOptions(ctx, path, data, opts.ABCIQueryOptions())
		if err != nil {
			// Retry on certain errors
			if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "invalid character '<' looking for beginning of value") {
				continue
			}

			// Return the error if it is not a retryable error
			return nil, err
		}

		// If the result is nil, return nil
		if result == nil {
			return nil, nil
		}

		// Return the response from the successful query
		return &result.Response, nil
	}

	// Return an error if the maximum retry limit is reached
	return nil, errors.New("reached max retry limit")
}

// QueryKey performs an ABCI query for a specific key in a store.
func (c *Context) QueryKey(ctx context.Context, store string, data bytes.HexBytes, opts *options.QueryOptions) (*abcitypes.ResponseQuery, error) {
	// Construct the path for the ABCI query
	path := fmt.Sprintf("/store/%s/key", store)

	// Call the generic ABCIQueryWithOptions method
	return c.ABCIQueryWithOptions(ctx, path, data, opts)
}

// QuerySubspace performs an ABCI query for a specific subspace in a store.
func (c *Context) QuerySubspace(ctx context.Context, store string, data bytes.HexBytes, opts *options.QueryOptions) (*abcitypes.ResponseQuery, error) {
	// Construct the path for the ABCI query
	path := fmt.Sprintf("/store/%s/subspace", store)

	// Call the generic ABCIQueryWithOptions method
	return c.ABCIQueryWithOptions(ctx, path, data, opts)
}
