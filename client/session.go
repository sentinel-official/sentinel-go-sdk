package client

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	hubtypes "github.com/sentinel-official/hub/types"
	sessiontypes "github.com/sentinel-official/hub/x/session/types"

	"github.com/sentinel-official/sentinel-go-sdk/v1/client/options"
)

// Session queries and returns information about a specific session based on the provided session ID.
// It uses gRPC to send a request to the "/sentinel.session.v2.QueryService/QuerySession" endpoint.
// The result is a pointer to sessiontypes.Session and an error if the query fails.
func (c *Context) Session(ctx context.Context, id uint64, opts *options.QueryOptions) (res *sessiontypes.Session, err error) {
	// Initialize variables for the query.
	var (
		resp   sessiontypes.QuerySessionResponse
		method = "/sentinel.session.v2.QueryService/QuerySession"
		req    = &sessiontypes.QuerySessionRequest{
			Id: id,
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return a pointer to the session and a nil error.
	return &resp.Session, nil
}

// Sessions queries and returns a list of sessions based on the provided options.
// It uses gRPC to send a request to the "/sentinel.session.v2.QueryService/QuerySessions" endpoint.
// The result is a slice of sessiontypes.Session and an error if the query fails.
func (c *Context) Sessions(ctx context.Context, opts *options.QueryOptions) (res []sessiontypes.Session, err error) {
	// Initialize variables for the query.
	var (
		resp   sessiontypes.QuerySessionsResponse
		method = "/sentinel.session.v2.QueryService/QuerySessions"
		req    = &sessiontypes.QuerySessionsRequest{
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of sessions and a nil error.
	return resp.Sessions, nil
}

// SessionsForAccount queries and returns a list of sessions associated with a specific account
// based on the provided account address and options.
// It uses gRPC to send a request to the "/sentinel.session.v2.QueryService/QuerySessionsForAccount" endpoint.
// The result is a slice of sessiontypes.Session and an error if the query fails.
func (c *Context) SessionsForAccount(ctx context.Context, accAddr sdk.AccAddress, opts *options.QueryOptions) (res []sessiontypes.Session, err error) {
	// Initialize variables for the query.
	var (
		resp   sessiontypes.QuerySessionsForAccountResponse
		method = "/sentinel.session.v2.QueryService/QuerySessionsForAccount"
		req    = &sessiontypes.QuerySessionsForAccountRequest{
			Address:    accAddr.String(),
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of sessions and a nil error.
	return resp.Sessions, nil
}

// SessionsForNode queries and returns a list of sessions associated with a specific node
// based on the provided node address and options.
// It uses gRPC to send a request to the "/sentinel.session.v2.QueryService/QuerySessionsForNode" endpoint.
// The result is a slice of sessiontypes.Session and an error if the query fails.
func (c *Context) SessionsForNode(ctx context.Context, nodeAddr hubtypes.NodeAddress, opts *options.QueryOptions) (res []sessiontypes.Session, err error) {
	// Initialize variables for the query.
	var (
		resp   sessiontypes.QuerySessionsForNodeResponse
		method = "/sentinel.session.v2.QueryService/QuerySessionsForNode"
		req    = &sessiontypes.QuerySessionsForNodeRequest{
			Address:    nodeAddr.String(),
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of sessions and a nil error.
	return resp.Sessions, nil
}

// SessionsForSubscription queries and returns a list of sessions associated with a specific subscription
// based on the provided subscription ID and options.
// It uses gRPC to send a request to the "/sentinel.session.v2.QueryService/QuerySessionsForSubscription" endpoint.
// The result is a slice of sessiontypes.Session and an error if the query fails.
func (c *Context) SessionsForSubscription(ctx context.Context, id uint64, opts *options.QueryOptions) (res []sessiontypes.Session, err error) {
	// Initialize variables for the query.
	var (
		resp   sessiontypes.QuerySessionsForSubscriptionResponse
		method = "/sentinel.session.v2.QueryService/QuerySessionsForSubscription"
		req    = &sessiontypes.QuerySessionsForSubscriptionRequest{
			Id:         id,
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of sessions and a nil error.
	return resp.Sessions, nil
}

// SessionsForSubscriptionAllocation queries and returns a list of sessions associated with a specific subscription allocation
// based on the provided subscription ID, account address, and options.
// It uses gRPC to send a request to the "/sentinel.session.v2.QueryService/QuerySessionsForAllocation" endpoint.
// The result is a slice of sessiontypes.Session and an error if the query fails.
func (c *Context) SessionsForSubscriptionAllocation(ctx context.Context, id uint64, accAddr sdk.AccAddress, opts *options.QueryOptions) (res []sessiontypes.Session, err error) {
	// Initialize variables for the query.
	var (
		resp   sessiontypes.QuerySessionsForAllocationResponse
		method = "/sentinel.session.v2.QueryService/QuerySessionsForAllocation"
		req    = &sessiontypes.QuerySessionsForAllocationRequest{
			Id:         id,
			Address:    accAddr.String(),
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of sessions and a nil error.
	return resp.Sessions, nil
}
