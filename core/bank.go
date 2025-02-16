package core

import (
	"context"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	// gRPC methods for querying bank balances
	methodQueryBalance  = "/cosmos.bank.v1beta1.Query/Balance"     // Retrieve the balance of a specific account
	methodQueryBalances = "/cosmos.bank.v1beta1.Query/AllBalances" // Retrieve all balances for a given account
)

// Balance retrieves the balance of a specific account and denomination.
// Returns the balance details and any error encountered.
func (c *Client) Balance(ctx context.Context, accAddr cosmossdk.AccAddress, denom string) (res *cosmossdk.Coin, err error) {
	var (
		resp bank.QueryBalanceResponse
		req  = &bank.QueryBalanceRequest{
			Address: accAddr.String(),
			Denom:   denom,
		}
	)

	// Perform the gRPC query to fetch the account balance.
	if err := c.QueryGRPC(ctx, methodQueryBalance, req, &resp); err != nil {
		return nil, IsCodeNotFound(err)
	}

	return resp.Balance, nil
}

// Balances retrieves all balances for a specific account with pagination.
// Returns the balances, pagination details, and any error encountered.
func (c *Client) Balances(ctx context.Context, accAddr cosmossdk.AccAddress, pageReq *query.PageRequest) (res cosmossdk.Coins, pageRes *query.PageResponse, err error) {
	var (
		resp bank.QueryAllBalancesResponse
		req  = &bank.QueryAllBalancesRequest{
			Address:    accAddr.String(),
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the account balances.
	if err := c.QueryGRPC(ctx, methodQueryBalances, req, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Balances, resp.Pagination, nil
}
