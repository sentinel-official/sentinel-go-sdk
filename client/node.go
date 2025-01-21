package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/types/query"
	sentinelhub "github.com/sentinel-official/hub/v12/types"
	"github.com/sentinel-official/hub/v12/types/v1"
	"github.com/sentinel-official/hub/v12/x/node/types/v3"

	"github.com/sentinel-official/sentinel-go-sdk/types"
)

const (
	// gRPC methods for querying node information
	methodQueryNode         = "/sentinel.node.v3.QueryService/QueryNode"         // Retrieve details of a specific node
	methodQueryNodes        = "/sentinel.node.v3.QueryService/QueryNodes"        // Retrieve a list of nodes with optional filtering
	methodQueryNodesForPlan = "/sentinel.node.v3.QueryService/QueryNodesForPlan" // Retrieve nodes associated with a specific plan
)

// Node retrieves details of a specific node by its address.
// Returns the node details and any error encountered.
func (c *Client) Node(ctx context.Context, nodeAddr sentinelhub.NodeAddress) (res *v3.Node, err error) {
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

// NodeInfo retrieves detailed information about a specific node by querying its remote URL.
func (c *Client) NodeInfo(ctx context.Context, nodeAddr sentinelhub.NodeAddress, target interface{}) error {
	// Query the node details.
	node, err := c.Node(ctx, nodeAddr)
	if err != nil {
		return fmt.Errorf("failed to query node: %w", err)
	}

	// Create a context with timeout for the HTTP request.
	ctx, cancel := context.WithTimeout(ctx, c.rpcTimeout)
	defer cancel()

	// Configure the HTTP client with TLS settings.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Perform the HTTP GET request.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, node.RemoteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get response: %w", err)
	}

	defer resp.Body.Close()

	// Check the response status code.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid response status code %d", resp.StatusCode)
	}

	// Decode the json response into the Response struct
	var body types.Response
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("failed to decode response body: %w", err)
	}

	// Check the response for success or error
	if err := body.Err(); err != nil {
		return err
	}

	// Decode the Result field into the provided target
	result, err := json.Marshal(body.Result)
	if err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}
	if err := json.Unmarshal(result, target); err != nil {
		return fmt.Errorf("failed to decode result: %w", err)
	}

	return nil
}
