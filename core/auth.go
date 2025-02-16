package core

import (
	"context"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	// gRPC methods for querying account information
	methodQueryAccount  = "/cosmos.auth.v1beta1.Query/Account"  // Endpoint for retrieving a single account
	methodQueryAccounts = "/cosmos.auth.v1beta1.Query/Accounts" // Endpoint for listing accounts with pagination
)

// Account retrieves an account by its address using a gRPC query.
// Returns the account interface and any potential error encountered.
func (c *Client) Account(ctx context.Context, accAddr cosmossdk.AccAddress) (res auth.AccountI, err error) {
	var (
		resp auth.QueryAccountResponse
		req  = &auth.QueryAccountRequest{Address: accAddr.String()}
	)

	// Perform the gRPC query to fetch the account details.
	if err := c.QueryGRPC(ctx, methodQueryAccount, req, &resp); err != nil {
		return nil, IsCodeNotFound(err)
	}

	// Unpack the retrieved account data into the account interface.
	if err := c.ProtoCodec().UnpackAny(resp.Account, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// Accounts retrieves a list of accounts with pagination support using a gRPC query.
// Returns a slice of account interfaces, pagination details, and any potential error.
func (c *Client) Accounts(ctx context.Context, pageReq *query.PageRequest) (res []auth.AccountI, pageRes *query.PageResponse, err error) {
	var (
		resp auth.QueryAccountsResponse
		req  = &auth.QueryAccountsRequest{Pagination: pageReq}
	)

	// Perform the gRPC query to fetch paginated account details.
	if err := c.QueryGRPC(ctx, methodQueryAccounts, req, &resp); err != nil {
		return nil, nil, err
	}

	// Allocate memory for account slice and unpack each account record.
	res = make([]auth.AccountI, len(resp.Accounts))
	for i := 0; i < len(resp.Accounts); i++ {
		if err := c.ProtoCodec().UnpackAny(resp.Accounts[i], &res[i]); err != nil {
			return nil, nil, err
		}
	}

	return res, resp.Pagination, nil
}
