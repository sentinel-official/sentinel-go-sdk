package core

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/tx"
)

const (
	// gRPC methods for simulating the transaction
	methodSimulate = "/cosmos.tx.v1beta1.Service/Simulate"
)

// Simulate simulates the execution of a transaction before broadcasting it.
// Takes transaction bytes as input and returns the simulation response or an error.
func (c *Client) Simulate(ctx context.Context, buf []byte) (*tx.SimulateResponse, error) {
	var (
		resp tx.SimulateResponse
		req  = &tx.SimulateRequest{TxBytes: buf}
	)

	// Perform a gRPC query to simulate the transaction.
	if err := c.QueryGRPC(ctx, methodSimulate, req, &resp); err != nil {
		return nil, HandleQueryErr(err)
	}

	return &resp, nil
}
