package client

import (
	"context"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	sentinelhub "github.com/sentinel-official/hub/types"
	subscriptiontypes "github.com/sentinel-official/hub/x/subscription/types"

	"github.com/sentinel-official/sentinel-go-sdk/v1/client/options"
)

// Subscription queries and returns information about a specific subscription based on the provided subscription ID.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QuerySubscription" endpoint.
// The result is a subscriptiontypes.Subscription and an error if the query fails.
func (c *Context) Subscription(ctx context.Context, id uint64, opts *options.QueryOptions) (res subscriptiontypes.Subscription, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QuerySubscriptionResponse
		method = "/sentinel.subscription.v2.QueryService/QuerySubscription"
		req    = &subscriptiontypes.QuerySubscriptionRequest{
			Id: id,
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Unpack the response and return the subscription and a nil error.
	if err := c.UnpackAny(resp.Subscription, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// Subscriptions queries and returns a list of subscriptions based on the provided options.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QuerySubscriptions" endpoint.
// The result is a slice of subscriptiontypes.Subscription and an error if the query fails.
func (c *Context) Subscriptions(ctx context.Context, opts *options.QueryOptions) (res []subscriptiontypes.Subscription, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QuerySubscriptionsResponse
		method = "/sentinel.subscription.v2.QueryService/QuerySubscriptions"
		req    = &subscriptiontypes.QuerySubscriptionsRequest{
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Unpack each subscription in the response and return the list of subscriptions and a nil error.
	res = make([]subscriptiontypes.Subscription, len(resp.Subscriptions))
	for i := 0; i < len(resp.Subscriptions); i++ {
		if err := c.UnpackAny(resp.Subscriptions[i], &res[i]); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// SubscriptionsForAccount queries and returns a list of subscriptions associated with a specific account.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QuerySubscriptionsForAccount" endpoint.
// The result is a slice of subscriptiontypes.Subscription and an error if the query fails.
// The account is identified by the provided cosmossdk.AccAddress.
func (c *Context) SubscriptionsForAccount(ctx context.Context, accAddr cosmossdk.AccAddress, opts *options.QueryOptions) (res []subscriptiontypes.Subscription, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QuerySubscriptionsForAccountResponse
		method = "/sentinel.subscription.v2.QueryService/QuerySubscriptionsForAccount"
		req    = &subscriptiontypes.QuerySubscriptionsForAccountRequest{
			Address:    accAddr.String(),
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Unpack each subscription in the response and return the list of subscriptions and a nil error.
	res = make([]subscriptiontypes.Subscription, len(resp.Subscriptions))
	for i := 0; i < len(resp.Subscriptions); i++ {
		if err := c.UnpackAny(resp.Subscriptions[i], &res[i]); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// SubscriptionsForNode queries and returns a list of subscriptions associated with a specific node.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QuerySubscriptionsForNode" endpoint.
// The result is a slice of subscriptiontypes.Subscription and an error if the query fails.
// The node is identified by the provided sentinelhub.NodeAddress.
func (c *Context) SubscriptionsForNode(ctx context.Context, nodeAddr sentinelhub.NodeAddress, opts *options.QueryOptions) (res []subscriptiontypes.Subscription, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QuerySubscriptionsForNodeResponse
		method = "/sentinel.subscription.v2.QueryService/QuerySubscriptionsForNode"
		req    = &subscriptiontypes.QuerySubscriptionsForNodeRequest{
			Address:    nodeAddr.String(),
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Unpack each subscription in the response and return the list of subscriptions and a nil error.
	res = make([]subscriptiontypes.Subscription, len(resp.Subscriptions))
	for i := 0; i < len(resp.Subscriptions); i++ {
		if err := c.UnpackAny(resp.Subscriptions[i], &res[i]); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// SubscriptionsForPlan queries and returns a list of subscriptions associated with a specific plan.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QuerySubscriptionsForPlan" endpoint.
// The result is a slice of subscriptiontypes.Subscription and an error if the query fails.
// The plan is identified by the provided ID.
func (c *Context) SubscriptionsForPlan(ctx context.Context, id uint64, opts *options.QueryOptions) (res []subscriptiontypes.Subscription, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QuerySubscriptionsForPlanResponse
		method = "/sentinel.subscription.v2.QueryService/QuerySubscriptionsForPlan"
		req    = &subscriptiontypes.QuerySubscriptionsForPlanRequest{
			Id:         id,
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Unpack each subscription in the response and return the list of subscriptions and a nil error.
	res = make([]subscriptiontypes.Subscription, len(resp.Subscriptions))
	for i := 0; i < len(resp.Subscriptions); i++ {
		if err := c.UnpackAny(resp.Subscriptions[i], &res[i]); err != nil {
			return nil, err
		}
	}

	return res, nil
}

// SubscriptionAllocation queries and returns information about a specific allocation within a subscription.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QueryAllocation" endpoint.
// The result is a pointer to subscriptiontypes.Allocation and an error if the query fails.
func (c *Context) SubscriptionAllocation(ctx context.Context, id uint64, accAddr cosmossdk.AccAddress, opts *options.QueryOptions) (res *subscriptiontypes.Allocation, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QueryAllocationResponse
		method = "/sentinel.subscription.v2.QueryService/QueryAllocation"
		req    = &subscriptiontypes.QueryAllocationRequest{
			Id:      id,
			Address: accAddr.String(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return a pointer to the allocation and a nil error.
	return &resp.Allocation, nil
}

// SubscriptionAllocations queries and returns a list of allocations within a specific subscription.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QueryAllocations" endpoint.
// The result is a slice of subscriptiontypes.Allocation and an error if the query fails.
func (c *Context) SubscriptionAllocations(ctx context.Context, id uint64, opts *options.QueryOptions) (res []subscriptiontypes.Allocation, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QueryAllocationsResponse
		method = "/sentinel.subscription.v2.QueryService/QueryAllocations"
		req    = &subscriptiontypes.QueryAllocationsRequest{
			Id:         id,
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of allocations and a nil error.
	return resp.Allocations, nil
}

// SubscriptionPayout queries and returns information about a specific payout within a subscription.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QueryPayout" endpoint.
// The result is a pointer to subscriptiontypes.Payout and an error if the query fails.
func (c *Context) SubscriptionPayout(ctx context.Context, id uint64, opts *options.QueryOptions) (res *subscriptiontypes.Payout, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QueryPayoutResponse
		method = "/sentinel.subscription.v2.QueryService/QueryPayout"
		req    = &subscriptiontypes.QueryPayoutRequest{
			Id: id,
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return a pointer to the payout and a nil error.
	return &resp.Payout, nil
}

// SubscriptionPayouts queries and returns a list of payouts within a specific subscription.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QueryPayouts" endpoint.
// The result is a slice of subscriptiontypes.Payout and an error if the query fails.
func (c *Context) SubscriptionPayouts(ctx context.Context, opts *options.QueryOptions) (res []subscriptiontypes.Payout, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QueryPayoutsResponse
		method = "/sentinel.subscription.v2.QueryService/QueryPayouts"
		req    = &subscriptiontypes.QueryPayoutsRequest{
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of payouts and a nil error.
	return resp.Payouts, nil
}

// SubscriptionPayoutsForAccount queries and returns a list of payouts associated with a specific account.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QueryPayoutsForAccount" endpoint.
// The result is a slice of subscriptiontypes.Payout and an error if the query fails.
// The account is identified by the provided cosmossdk.AccAddress.
func (c *Context) SubscriptionPayoutsForAccount(ctx context.Context, accAddr cosmossdk.AccAddress, opts *options.QueryOptions) (res []subscriptiontypes.Payout, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QueryPayoutsForAccountResponse
		method = "/sentinel.subscription.v2.QueryService/QueryPayoutsForAccount"
		req    = &subscriptiontypes.QueryPayoutsForAccountRequest{
			Address:    accAddr.String(),
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of payouts and a nil error.
	return resp.Payouts, nil
}

// SubscriptionPayoutsForNode queries and returns a list of payouts associated with a specific node.
// It uses gRPC to send a request to the "/sentinel.subscription.v2.QueryService/QueryPayoutsForNode" endpoint.
// The result is a slice of subscriptiontypes.Payout and an error if the query fails.
// The node is identified by the provided sentinelhub.NodeAddress.
func (c *Context) SubscriptionPayoutsForNode(ctx context.Context, nodeAddr sentinelhub.NodeAddress, opts *options.QueryOptions) (res []subscriptiontypes.Payout, err error) {
	// Initialize variables for the query.
	var (
		resp   subscriptiontypes.QueryPayoutsForNodeResponse
		method = "/sentinel.subscription.v2.QueryService/QueryPayoutsForNode"
		req    = &subscriptiontypes.QueryPayoutsForNodeRequest{
			Address:    nodeAddr.String(),
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of payouts and a nil error.
	return resp.Payouts, nil
}
