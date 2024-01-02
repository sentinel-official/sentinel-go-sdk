package client

import (
	"context"

	sentinelhub "github.com/sentinel-official/hub/types"
	plantypes "github.com/sentinel-official/hub/x/plan/types"

	"github.com/sentinel-official/sentinel-go-sdk/v1/client/options"
)

// Plan queries and returns information about a specific plan based on the provided plan ID.
// It uses gRPC to send a request to the "/sentinel.plan.v2.QueryService/QueryPlan" endpoint.
// The result is a pointer to plantypes.Plan and an error if the query fails.
func (c *Context) Plan(ctx context.Context, id uint64, opts *options.QueryOptions) (res *plantypes.Plan, err error) {
	// Initialize variables for the query.
	var (
		resp   plantypes.QueryPlanResponse
		method = "/sentinel.plan.v2.QueryService/QueryPlan"
		req    = &plantypes.QueryPlanRequest{
			Id: id,
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return a pointer to the plan and a nil error.
	return &resp.Plan, nil
}

// Plans queries and returns a list of plans based on the provided status and options.
// It uses gRPC to send a request to the "/sentinel.plan.v2.QueryService/QueryPlans" endpoint.
// The result is a slice of plantypes.Plan and an error if the query fails.
func (c *Context) Plans(ctx context.Context, status sentinelhub.Status, opts *options.QueryOptions) (res []plantypes.Plan, err error) {
	// Initialize variables for the query.
	var (
		resp   plantypes.QueryPlansResponse
		method = "/sentinel.plan.v2.QueryService/QueryPlans"
		req    = &plantypes.QueryPlansRequest{
			Status:     status,
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of plans and a nil error.
	return resp.Plans, nil
}

// PlansForProvider queries and returns a list of plans associated with a specific provider
// based on the provided provider address, status, and options.
// It uses gRPC to send a request to the "/sentinel.plan.v2.QueryService/QueryPlansForProvider" endpoint.
// The result is a slice of plantypes.Plan and an error if the query fails.
func (c *Context) PlansForProvider(ctx context.Context, provAddr sentinelhub.ProvAddress, status sentinelhub.Status, opts *options.QueryOptions) (res []plantypes.Plan, err error) {
	// Initialize variables for the query.
	var (
		resp   plantypes.QueryPlansForProviderResponse
		method = "/sentinel.plan.v2.QueryService/QueryPlansForProvider"
		req    = &plantypes.QueryPlansForProviderRequest{
			Address:    provAddr.String(),
			Status:     status,
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of plans and a nil error.
	return resp.Plans, nil
}
