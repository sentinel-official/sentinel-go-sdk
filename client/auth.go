package client

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/sentinel-official/sentinel-sdk/v1/client/options"
)

// Account queries the account information for a given account address.
func (c *Context) Account(ctx context.Context, accAddr sdk.AccAddress, opts *options.QueryOptions) (res authtypes.AccountI, err error) {
	// Query the account information using the generic QueryKey method
	resp, err := c.QueryKey(ctx, authtypes.StoreKey, authtypes.AddressStoreKey(accAddr), opts)
	if err != nil {
		return nil, err
	}

	// If the response is nil, return nil
	if resp == nil {
		return nil, nil
	}

	// Unmarshal the response value into an authtypes.AccountI interface
	if err := c.UnmarshalInterface(resp.Value, &res); err != nil {
		return nil, err
	}

	// Return the unmarshalled account information
	return res, nil
}
