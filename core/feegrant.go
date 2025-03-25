package core

import (
	"context"
	"strings"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

const (
	// gRPC methods for querying fee grants
	methodQueryFeegrantAllowance           = "/cosmos.feegrant.v1beta1.Query/Allowance"           // Retrieve details of a specific fee grant
	methodQueryFeegrantAllowances          = "/cosmos.feegrant.v1beta1.Query/Allowances"          // Retrieve a list of fee grants
	methodQueryFeegrantAllowancesByGranter = "/cosmos.feegrant.v1beta1.Query/AllowancesByGranter" // Retrieve a list of fee grants issued by a specific granter
)

// FeegrantAllowance retrieves the fee grant allowance given by a granter to a grantee.
// Returns the fee grant details and any error encountered.
func (c *Client) FeegrantAllowance(ctx context.Context, granter, grantee types.AccAddress) (res *feegrant.Grant, err error) {
	var (
		resp feegrant.QueryAllowanceResponse
		req  = &feegrant.QueryAllowanceRequest{
			Granter: granter.String(),
			Grantee: grantee.String(),
		}
	)

	// Perform the gRPC query to fetch the fee grant allowance.
	if err := c.QueryGRPC(ctx, methodQueryFeegrantAllowance, req, &resp); err != nil {
		if strings.Contains(err.Error(), "fee-grant not found") {
			return nil, nil
		}

		return nil, err
	}

	return resp.Allowance, nil
}

// FeegrantAllowances retrieves a paginated list of fee grants for a given grantee.
// Returns the list of fee grants, pagination details, and any error encountered.
func (c *Client) FeegrantAllowances(ctx context.Context, grantee types.AccAddress, pageReq *query.PageRequest) (res []*feegrant.Grant, pageRes *query.PageResponse, err error) {
	var (
		resp feegrant.QueryAllowancesResponse
		req  = &feegrant.QueryAllowancesRequest{
			Grantee:    grantee.String(),
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the fee grants for the given grantee.
	if err := c.QueryGRPC(ctx, methodQueryFeegrantAllowances, req, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Allowances, resp.Pagination, nil
}

// FeegrantAllowancesByGranter retrieves a paginated list of fee grants issued by a specific granter.
// Returns the list of fee grants, pagination details, and any error encountered.
func (c *Client) FeegrantAllowancesByGranter(ctx context.Context, granter types.AccAddress, pageReq *query.PageRequest) (res []*feegrant.Grant, pageRes *query.PageResponse, err error) {
	var (
		resp feegrant.QueryAllowancesByGranterResponse
		req  = &feegrant.QueryAllowancesByGranterRequest{
			Granter:    granter.String(),
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the fee grants issued by the specified granter.
	if err := c.QueryGRPC(ctx, methodQueryFeegrantAllowancesByGranter, req, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Allowances, resp.Pagination, nil
}
