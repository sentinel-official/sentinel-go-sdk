package core

import (
	"context"
	"fmt"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const (
	// gRPC methods for querying account information.
	methodQueryAccount  = "/cosmos.auth.v1beta1.Query/Account"  // Endpoint for retrieving a single account
	methodQueryAccounts = "/cosmos.auth.v1beta1.Query/Accounts" // Endpoint for listing accounts with pagination
)

// Account retrieves an account by its address.
// Returns the account interface and any potential error encountered.
func (c *Client) Account(ctx context.Context, accAddr cosmossdk.AccAddress) (res auth.AccountI, err error) {
	var (
		resp auth.QueryAccountResponse
		req  = &auth.QueryAccountRequest{Address: accAddr.String()}
	)

	// Perform the ABCI query to fetch the account details.
	if err := c.QueryABCI(ctx, methodQueryAccount, req, &resp); err != nil {
		return nil, HandleQueryErr(err)
	}

	// Unpack the retrieved account data into the account interface.
	if err := c.ProtoCodec().UnpackAny(resp.Account, &res); err != nil {
		return nil, fmt.Errorf("unpacking account: %w", err)
	}

	return res, nil
}

// Accounts retrieves a list of accounts with pagination support.
// Returns a slice of account interfaces, pagination details, and any potential error.
func (c *Client) Accounts(ctx context.Context, pageReq *query.PageRequest) (res []auth.AccountI, pageRes *query.PageResponse, err error) {
	var (
		resp auth.QueryAccountsResponse
		req  = &auth.QueryAccountsRequest{Pagination: pageReq}
	)

	// Perform the ABCI query to fetch paginated account details.
	if err := c.QueryABCI(ctx, methodQueryAccounts, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	// Allocate memory for account slice and unpack each account record.
	res = make([]auth.AccountI, len(resp.Accounts))
	for i := range len(resp.Accounts) {
		if err := c.ProtoCodec().UnpackAny(resp.Accounts[i], &res[i]); err != nil {
			return nil, nil, fmt.Errorf("unpacking account: %w", err)
		}
	}

	return res, resp.Pagination, nil
}
