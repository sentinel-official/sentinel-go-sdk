package core

import (
	"context"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sentinel-official/sentinelhub/v12/x/subscription/types/v2"
	"github.com/sentinel-official/sentinelhub/v12/x/subscription/types/v3"
)

const (
	// gRPC methods for querying subscription and allocation information.
	methodQuerySubscription            = "/sentinel.subscription.v3.QueryService/QuerySubscription"            // Fetch details of a specific subscription
	methodQuerySubscriptionAllocation  = "/sentinel.subscription.v2.QueryService/QueryAllocation"              // Fetch details of a specific allocation within a subscription
	methodQuerySubscriptionAllocations = "/sentinel.subscription.v2.QueryService/QueryAllocations"             // Fetch a list of allocations within a subscription
	methodQuerySubscriptionParams      = "/sentinel.subscription.v3.QueryService/QueryParams"                  // Retrieve module parameters for subscriptions
	methodQuerySubscriptions           = "/sentinel.subscription.v3.QueryService/QuerySubscriptions"           // Fetch a list of all subscriptions
	methodQuerySubscriptionsForAccount = "/sentinel.subscription.v3.QueryService/QuerySubscriptionsForAccount" // Fetch subscriptions associated with an account
	methodQuerySubscriptionsForPlan    = "/sentinel.subscription.v3.QueryService/QuerySubscriptionsForPlan"    // Fetch subscriptions associated with a specific plan
)

// Subscription retrieves details of a specific subscription by its ID.
// Returns the subscription details and any error encountered.
func (c *Client) Subscription(ctx context.Context, id uint64) (res *v3.Subscription, err error) {
	var (
		resp v3.QuerySubscriptionResponse
		req  = &v3.QuerySubscriptionRequest{Id: id}
	)

	// Perform the ABCI query to fetch the subscription details.
	if err := c.QueryABCI(ctx, methodQuerySubscription, req, &resp); err != nil {
		return nil, HandleQueryErr(err)
	}

	return &resp.Subscription, nil
}

// SubscriptionAllocation retrieves details of a specific allocation within a subscription.
// Returns the allocation details and any error encountered.
func (c *Client) SubscriptionAllocation(ctx context.Context, id uint64, accAddr cosmossdk.AccAddress) (res *v2.Allocation, err error) {
	var (
		resp v2.QueryAllocationResponse
		req  = &v2.QueryAllocationRequest{
			Id:      id,
			Address: accAddr.String(),
		}
	)

	// Perform the ABCI query to fetch the allocation details.
	if err := c.QueryABCI(ctx, methodQuerySubscriptionAllocation, req, &resp); err != nil {
		return nil, HandleQueryErr(err)
	}

	return &resp.Allocation, nil
}

// SubscriptionAllocations retrieves a paginated list of allocations within a specific subscription.
// Returns the allocations, pagination details, and any error encountered.
func (c *Client) SubscriptionAllocations(ctx context.Context, id uint64, pageReq *query.PageRequest) (res []v2.Allocation, pageRes *query.PageResponse, err error) {
	var (
		resp v2.QueryAllocationsResponse
		req  = &v2.QueryAllocationsRequest{
			Id:         id,
			Pagination: pageReq,
		}
	)

	// Perform the ABCI query to fetch the allocations.
	if err := c.QueryABCI(ctx, methodQuerySubscriptionAllocations, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	return resp.Allocations, resp.Pagination, nil
}

// SubscriptionParams retrieves the current parameters for the subscription module.
// Returns the subscription parameters and any error encountered.
func (c *Client) SubscriptionParams(ctx context.Context) (res *v3.Params, err error) {
	var (
		resp v3.QueryParamsResponse
		req  = &v3.QueryParamsRequest{}
	)

	// Perform the ABCI query to fetch the subscription module parameters.
	if err := c.QueryABCI(ctx, methodQuerySubscriptionParams, req, &resp); err != nil {
		return nil, HandleQueryErr(err)
	}

	return &resp.Params, nil
}

// Subscriptions retrieves a paginated list of all subscriptions.
// Returns the subscriptions, pagination details, and any error encountered.
func (c *Client) Subscriptions(ctx context.Context, pageReq *query.PageRequest) (res []v3.Subscription, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QuerySubscriptionsResponse
		req  = &v3.QuerySubscriptionsRequest{Pagination: pageReq}
	)

	// Perform the ABCI query to fetch the subscriptions.
	if err := c.QueryABCI(ctx, methodQuerySubscriptions, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	return resp.Subscriptions, resp.Pagination, nil
}

// SubscriptionsForAccount retrieves subscriptions associated with a specific account.
// Returns the subscriptions, pagination details, and any error encountered.
func (c *Client) SubscriptionsForAccount(ctx context.Context, accAddr cosmossdk.AccAddress, pageReq *query.PageRequest) (res []v3.Subscription, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QuerySubscriptionsForAccountResponse
		req  = &v3.QuerySubscriptionsForAccountRequest{
			Address:    accAddr.String(),
			Pagination: pageReq,
		}
	)

	// Perform the ABCI query to fetch subscriptions for the given account.
	if err := c.QueryABCI(ctx, methodQuerySubscriptionsForAccount, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	return resp.Subscriptions, resp.Pagination, nil
}

// SubscriptionsForPlan retrieves subscriptions associated with a specific plan.
// Returns the subscriptions, pagination details, and any error encountered.
func (c *Client) SubscriptionsForPlan(ctx context.Context, id uint64, pageReq *query.PageRequest) (res []v3.Subscription, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QuerySubscriptionsForPlanResponse
		req  = &v3.QuerySubscriptionsForPlanRequest{
			Id:         id,
			Pagination: pageReq,
		}
	)

	// Perform the ABCI query to fetch subscriptions for the given plan.
	if err := c.QueryABCI(ctx, methodQuerySubscriptionsForPlan, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	return resp.Subscriptions, resp.Pagination, nil
}
