package core

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sentinel-official/sentinelhub/v12/types"
	"github.com/sentinel-official/sentinelhub/v12/types/v1"
	"github.com/sentinel-official/sentinelhub/v12/x/plan/types/v3"
)

const (
	// gRPC methods for querying plan information.
	methodQueryPlan             = "/sentinel.plan.v3.QueryService/QueryPlan"             // Retrieve details of a specific plan
	methodQueryPlans            = "/sentinel.plan.v3.QueryService/QueryPlans"            // Retrieve a list of plans
	methodQueryPlansForProvider = "/sentinel.plan.v3.QueryService/QueryPlansForProvider" // Retrieve plans associated with a specific provider
)

// Plan retrieves details of a specific plan by its ID.
// Returns the plan details and any error encountered.
func (c *Client) Plan(ctx context.Context, id uint64) (res *v3.Plan, err error) {
	var (
		resp v3.QueryPlanResponse
		req  = &v3.QueryPlanRequest{Id: id}
	)

	// Perform the gRPC query to fetch the plan details.
	if err := c.QueryGRPC(ctx, methodQueryPlan, req, &resp); err != nil {
		return nil, HandleQueryErr(err)
	}

	return &resp.Plan, nil
}

// Plans retrieves a paginated list of plans filtered by their status.
// Returns the plans, pagination details, and any error encountered.
func (c *Client) Plans(ctx context.Context, status v1.Status, pageReq *query.PageRequest) (res []v3.Plan, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QueryPlansResponse
		req  = &v3.QueryPlansRequest{
			Status:     status,
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the plans.
	if err := c.QueryGRPC(ctx, methodQueryPlans, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	return resp.Plans, resp.Pagination, nil
}

// PlansForProvider retrieves a list of plans associated with a specific provider address.
// Filters results by status and supports pagination.
// Returns the plans, pagination details, and any error encountered.
func (c *Client) PlansForProvider(ctx context.Context, provAddr types.ProvAddress, status v1.Status, pageReq *query.PageRequest) (res []v3.Plan, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QueryPlansForProviderResponse
		req  = &v3.QueryPlansForProviderRequest{
			Address:    provAddr.String(),
			Status:     status,
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch plans for the given provider.
	if err := c.QueryGRPC(ctx, methodQueryPlansForProvider, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	return resp.Plans, resp.Pagination, nil
}
