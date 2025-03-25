package core

import (
	"context"
	"strings"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

const (
	// gRPC methods for querying authorization grants
	methodQueryAuthzGranteeGrants = "/cosmos.authz.v1beta1.Query/GranteeGrants" // Retrieve a list of grants for a specific grantee
	methodQueryAuthzGranterGrants = "/cosmos.authz.v1beta1.Query/GranterGrants" // Retrieve a list of grants issued by a specific granter
	methodQueryAuthzGrants        = "/cosmos.authz.v1beta1.Query/Grants"        // Retrieve details of a specific grant
)

// AuthzGranteeGrants retrieves a paginated list of grants assigned to a specific grantee.
// Returns the list of grants, pagination details, and any error encountered.
func (c *Client) AuthzGranteeGrants(ctx context.Context, grantee types.AccAddress, pageReq *query.PageRequest) (res []*authz.GrantAuthorization, pageRes *query.PageResponse, err error) {
	var (
		resp authz.QueryGranteeGrantsResponse
		req  = &authz.QueryGranteeGrantsRequest{
			Grantee:    grantee.String(),
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the grants assigned to the specified grantee.
	if err := c.QueryGRPC(ctx, methodQueryAuthzGranteeGrants, req, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Grants, resp.Pagination, nil
}

// AuthzGranterGrants retrieves a paginated list of grants issued by a specific granter.
// Returns the list of grants, pagination details, and any error encountered.
func (c *Client) AuthzGranterGrants(ctx context.Context, granter types.AccAddress, pageReq *query.PageRequest) (res []*authz.GrantAuthorization, pageRes *query.PageResponse, err error) {
	var (
		resp authz.QueryGranterGrantsResponse
		req  = &authz.QueryGranterGrantsRequest{
			Granter:    granter.String(),
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the grants issued by the specified granter.
	if err := c.QueryGRPC(ctx, methodQueryAuthzGranterGrants, req, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Grants, resp.Pagination, nil
}

// AuthzGrants retrieves a paginated list of grants for a specific granter and grantee combination.
// Returns the list of grants, pagination details, and any error encountered.
func (c *Client) AuthzGrants(ctx context.Context, granter, grantee types.AccAddress, msgTypeURL string, pageReq *query.PageRequest) (res []*authz.Grant, pageRes *query.PageResponse, err error) {
	var (
		resp authz.QueryGrantsResponse
		req  = &authz.QueryGrantsRequest{
			Granter:    granter.String(),
			Grantee:    grantee.String(),
			MsgTypeUrl: msgTypeURL,
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the grants for the specified granter and grantee.
	if err := c.QueryGRPC(ctx, methodQueryAuthzGrants, req, &resp); err != nil {
		if strings.Contains(err.Error(), authz.ErrNoAuthorizationFound.Error()) {
			return nil, nil, nil
		}

		return nil, nil, err
	}

	return resp.Grants, resp.Pagination, nil
}
