package core

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sentinel-official/sentinelhub/v12/x/deposit/types/v1"
)

const (
	// gRPC methods for querying deposit information
	methodQueryDeposit  = "/sentinel.deposit.v1.QueryService/QueryDeposit"  // Retrieve details of a specific deposit
	methodQueryDeposits = "/sentinel.deposit.v1.QueryService/QueryDeposits" // Retrieve a list of deposits
)

// Deposit retrieves details of a specific deposit by its address.
// Returns the deposit details and any error encountered.
func (c *Client) Deposit(ctx context.Context, accAddr types.AccAddress) (res *v1.Deposit, err error) {
	var (
		resp v1.QueryDepositResponse
		req  = &v1.QueryDepositRequest{Address: accAddr.String()}
	)

	// Perform the gRPC query to fetch the deposit details.
	if err := c.QueryGRPC(ctx, methodQueryDeposit, req, &resp); err != nil {
		return nil, IsCodeNotFound(err)
	}

	return &resp.Deposit, nil
}

// Deposits retrieves a paginated list of deposits.
// Returns the deposits, pagination details, and any error encountered.
func (c *Client) Deposits(ctx context.Context, pageReq *query.PageRequest) (res []v1.Deposit, pageRes *query.PageResponse, err error) {
	var (
		resp v1.QueryDepositsResponse
		req  = &v1.QueryDepositsRequest{
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the deposits.
	if err := c.QueryGRPC(ctx, methodQueryDeposits, req, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Deposits, resp.Pagination, nil
}
