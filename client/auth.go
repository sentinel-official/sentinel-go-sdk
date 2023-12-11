package client

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/sentinel-official/sentinel-sdk/v1/client/options"
)

// Account queries and returns an account using the given address and options.
// It uses gRPC to send a request to the "/cosmos.auth.v1beta1.Query/Account" endpoint.
// The result is an authtypes.AccountI interface and an error if the query fails.
func (c *Context) Account(ctx context.Context, accAddr sdk.AccAddress, opts *options.QueryOptions) (res authtypes.AccountI, err error) {
	// Initialize variables for the query.
	var (
		resp   authtypes.QueryAccountResponse
		method = "/cosmos.auth.v1beta1.Query/Account"
		req    = &authtypes.QueryAccountRequest{
			Address: accAddr.String(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Unpack the Any type account from the response.
	if err := c.UnpackAny(resp.Account, &res); err != nil {
		return nil, err
	}

	// Return the account and a nil error.
	return res, nil
}

// Accounts queries and returns a list of accounts using the given options.
// It uses gRPC to send a request to the "/cosmos.auth.v1beta1.Query/Accounts" endpoint.
// The result is a slice of authtypes.AccountI and an error if the query fails.
func (c *Context) Accounts(ctx context.Context, opts *options.QueryOptions) (res []authtypes.AccountI, err error) {
	// Initialize variables for the query.
	var (
		resp   authtypes.QueryAccountsResponse
		method = "/cosmos.auth.v1beta1.Query/Accounts"
		req    = &authtypes.QueryAccountsRequest{
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Initialize a slice to store the accounts.
	res = make([]authtypes.AccountI, len(resp.Accounts))

	// Unpack each Any type account from the response and add it to the result slice.
	for i := 0; i < len(resp.Accounts); i++ {
		if err := c.UnpackAny(resp.Accounts[i], &res[i]); err != nil {
			return nil, err
		}
	}

	// Return the list of accounts and a nil error.
	return res, nil
}
