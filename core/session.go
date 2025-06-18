package core

import (
	"context"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	sentinelhub "github.com/sentinel-official/sentinelhub/v12/types"
	"github.com/sentinel-official/sentinelhub/v12/x/session/types/v3"
)

const (
	// gRPC methods for querying session information
	methodQuerySession                           = "/sentinel.session.v3.QueryService/QuerySession"                 // Retrieve details of a specific session
	methodQuerySessions                          = "/sentinel.session.v3.QueryService/QuerySessions"                // Retrieve a list of sessions with pagination
	methodQuerySessionsForAccount                = "/sentinel.session.v3.QueryService/QuerySessionsForAccount"      // Retrieve sessions associated with a specific account
	methodQuerySessionsForNode                   = "/sentinel.session.v3.QueryService/QuerySessionsForNode"         // Retrieve sessions associated with a specific node
	methodQuerySessionsForSubscription           = "/sentinel.session.v3.QueryService/QuerySessionsForSubscription" // Retrieve sessions associated with a specific subscription
	methodQuerySessionsForSubscriptionAllocation = "/sentinel.session.v3.QueryService/QuerySessionsForAllocation"   // Retrieve sessions for a subscription and account
)

// Session retrieves details of a specific session by its ID.
// Returns the session details and any error encountered.
func (c *Client) Session(ctx context.Context, id uint64) (res v3.Session, err error) {
	var (
		resp v3.QuerySessionResponse
		req  = &v3.QuerySessionRequest{Id: id}
	)

	// Perform the gRPC query to fetch the session details.
	if err := c.QueryGRPC(ctx, methodQuerySession, req, &resp); err != nil {
		return nil, IsCodeNotFound(err)
	}

	// Unpack the session data from the response.
	if err := c.ProtoCodec().UnpackAny(resp.Session, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// Sessions retrieves a paginated list of all sessions.
// Returns the sessions, pagination details, and any error encountered.
func (c *Client) Sessions(ctx context.Context, pageReq *query.PageRequest) (res []v3.Session, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QuerySessionsResponse
		req  = &v3.QuerySessionsRequest{Pagination: pageReq}
	)

	// Perform the gRPC query to fetch the sessions.
	if err := c.QueryGRPC(ctx, methodQuerySessions, req, &resp); err != nil {
		return nil, nil, err
	}

	// Unpack each session from the response.
	res = make([]v3.Session, len(resp.Sessions))
	for i := 0; i < len(resp.Sessions); i++ {
		if err := c.ProtoCodec().UnpackAny(resp.Sessions[i], &res[i]); err != nil {
			return nil, nil, err
		}
	}

	return res, resp.Pagination, nil
}

// SessionsForAccount retrieves sessions associated with a specific account address.
// Returns the sessions, pagination details, and any error encountered.
func (c *Client) SessionsForAccount(ctx context.Context, accAddr cosmossdk.AccAddress, pageReq *query.PageRequest) (res []v3.Session, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QuerySessionsForAccountResponse
		req  = &v3.QuerySessionsForAccountRequest{
			Address:    accAddr.String(),
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch sessions for the given account.
	if err := c.QueryGRPC(ctx, methodQuerySessionsForAccount, req, &resp); err != nil {
		return nil, nil, err
	}

	// Unpack each session from the response.
	res = make([]v3.Session, len(resp.Sessions))
	for i := 0; i < len(resp.Sessions); i++ {
		if err := c.ProtoCodec().UnpackAny(resp.Sessions[i], &res[i]); err != nil {
			return nil, nil, err
		}
	}

	return res, resp.Pagination, nil
}

// SessionsForNode retrieves sessions associated with a specific node address.
// Returns the sessions, pagination details, and any error encountered.
func (c *Client) SessionsForNode(ctx context.Context, nodeAddr sentinelhub.NodeAddress, pageReq *query.PageRequest) (res []v3.Session, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QuerySessionsForNodeResponse
		req  = &v3.QuerySessionsForNodeRequest{
			Address:    nodeAddr.String(),
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch sessions for the given node.
	if err := c.QueryGRPC(ctx, methodQuerySessionsForNode, req, &resp); err != nil {
		return nil, nil, err
	}

	// Unpack each session from the response.
	res = make([]v3.Session, len(resp.Sessions))
	for i := 0; i < len(resp.Sessions); i++ {
		if err := c.ProtoCodec().UnpackAny(resp.Sessions[i], &res[i]); err != nil {
			return nil, nil, err
		}
	}

	return res, resp.Pagination, nil
}

// SessionsForSubscription retrieves sessions associated with a specific subscription ID.
// Returns the sessions, pagination details, and any error encountered.
func (c *Client) SessionsForSubscription(ctx context.Context, id uint64, pageReq *query.PageRequest) (res []v3.Session, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QuerySessionsForSubscriptionResponse
		req  = &v3.QuerySessionsForSubscriptionRequest{
			Id:         id,
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch sessions for the given subscription.
	if err := c.QueryGRPC(ctx, methodQuerySessionsForSubscription, req, &resp); err != nil {
		return nil, nil, err
	}

	// Unpack each session from the response.
	res = make([]v3.Session, len(resp.Sessions))
	for i := 0; i < len(resp.Sessions); i++ {
		if err := c.ProtoCodec().UnpackAny(resp.Sessions[i], &res[i]); err != nil {
			return nil, nil, err
		}
	}

	return res, resp.Pagination, nil
}

// SessionsForSubscriptionAllocation retrieves sessions associated with a specific subscription ID and account address.
// Returns the sessions, pagination details, and any error encountered.
func (c *Client) SessionsForSubscriptionAllocation(ctx context.Context, id uint64, accAddr cosmossdk.AccAddress, pageReq *query.PageRequest) (res []v3.Session, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QuerySessionsForAllocationResponse
		req  = &v3.QuerySessionsForAllocationRequest{
			Id:         id,
			Address:    accAddr.String(),
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch sessions for the given subscription and account.
	if err := c.QueryGRPC(ctx, methodQuerySessionsForSubscriptionAllocation, req, &resp); err != nil {
		return nil, nil, err
	}

	// Unpack each session from the response.
	res = make([]v3.Session, len(resp.Sessions))
	for i := 0; i < len(resp.Sessions); i++ {
		if err := c.ProtoCodec().UnpackAny(resp.Sessions[i], &res[i]); err != nil {
			return nil, nil, err
		}
	}

	return res, resp.Pagination, nil
}
