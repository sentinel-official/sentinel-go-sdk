package core

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sentinel-official/hub/v12/types"
	"github.com/sentinel-official/hub/v12/types/v1"
	"github.com/sentinel-official/hub/v12/x/node/types/v3"
)

const (
	// gRPC methods for querying node information
	methodQueryNode         = "/sentinel.node.v3.QueryService/QueryNode"         // Retrieve details of a specific node
	methodQueryNodes        = "/sentinel.node.v3.QueryService/QueryNodes"        // Retrieve a list of nodes with optional filtering
	methodQueryNodesForPlan = "/sentinel.node.v3.QueryService/QueryNodesForPlan" // Retrieve nodes associated with a specific plan
)

// Node retrieves details of a specific node by its address.
// Returns the node details and any error encountered.
func (c *Client) Node(ctx context.Context, nodeAddr types.NodeAddress) (res *v3.Node, err error) {
	var (
		resp v3.QueryNodeResponse
		req  = &v3.QueryNodeRequest{Address: nodeAddr.String()}
	)

	// Perform the gRPC query to fetch the node details.
	if err := c.QueryGRPC(ctx, methodQueryNode, req, &resp); err != nil {
		return nil, IsNotFoundError(err)
	}

	return &resp.Node, nil
}

// Nodes retrieves a paginated list of nodes filtered by their status.
// Returns the nodes, pagination details, and any error encountered.
func (c *Client) Nodes(ctx context.Context, status v1.Status, pageReq *query.PageRequest) (res []v3.Node, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QueryNodesResponse
		req  = &v3.QueryNodesRequest{
			Status:     status,
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the nodes.
	if err := c.QueryGRPC(ctx, methodQueryNodes, req, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Nodes, resp.Pagination, nil
}

// NodesForPlan retrieves a list of nodes associated with a specific plan ID.
// Filters results by status and supports pagination.
// Returns the nodes, pagination details, and any error encountered.
func (c *Client) NodesForPlan(ctx context.Context, id uint64, status v1.Status, pageReq *query.PageRequest) (res []v3.Node, pageRes *query.PageResponse, err error) {
	var (
		resp v3.QueryNodesForPlanResponse
		req  = &v3.QueryNodesForPlanRequest{
			Id:         id,
			Status:     status,
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch nodes for the given plan.
	if err := c.QueryGRPC(ctx, methodQueryNodesForPlan, req, &resp); err != nil {
		return nil, nil, err
	}

	return resp.Nodes, resp.Pagination, nil
}
