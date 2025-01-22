package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

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
func (c *BaseClient) Node(ctx context.Context, nodeAddr sentinelhub.NodeAddress) (res *v3.Node, err error) {
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
func (c *BaseClient) Nodes(ctx context.Context, status v1.Status, pageReq *query.PageRequest) (res []v3.Node, pageRes *query.PageResponse, err error) {
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
func (c *BaseClient) NodesForPlan(ctx context.Context, id uint64, status v1.Status, pageReq *query.PageRequest) (res []v3.Node, pageRes *query.PageResponse, err error) {
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

// do performs an HTTP request with the given parameters and decodes the response.
func (c *NodeClient) do(ctx context.Context, method, url string, reqBody, result interface{}) error {
	// Create a context with timeout for the HTTP request.
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Configure the HTTP client with TLS settings.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Marshal the request body if provided.
	var body io.Reader
	if reqBody != nil {
		buf, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}

		body = bytes.NewReader(buf)
	}

	// Create the HTTP request.
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Perform the HTTP request.
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request: %w", err)
	}

	defer resp.Body.Close()

	// Check for a successful status code.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status code: %d", resp.StatusCode)
	}

	// Decode the JSON response into a predefined structure.
	var respBody types.Response
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return fmt.Errorf("failed to decode response body: %w", err)
	}

	// Check for errors in the response.
	if err := respBody.Err(); err != nil {
		return fmt.Errorf("response error: %w", err)
	}

	// Decode the Result field if a result target is provided.
	if result != nil {
		buf, err := json.Marshal(respBody.Result)
		if err != nil {
			return fmt.Errorf("failed to encode data: %w", err)
		}
		if err := json.Unmarshal(buf, result); err != nil {
			return fmt.Errorf("failed to decode result: %w", err)
		}
	}

	return nil
}

// getURL constructs the full URL for a node with an optional path.
func (c *NodeClient) getURL(ctx context.Context, pathSuffix string) (string, error) {
	node, err := c.Node(ctx, c.addr)
	if err != nil {
		return "", fmt.Errorf("failed to query node: %w", err)
	}

	path, err := url.JoinPath(node.RemoteURL, pathSuffix)
	if err != nil {
		return "", fmt.Errorf("failed to join url path: %w", err)
	}

	return path, nil
}

// GetInfo retrieves information about a specific node.
func (c *NodeClient) GetInfo(ctx context.Context, result interface{}) error {
	path, err := c.getURL(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get url: %w", err)
	}

	return c.do(ctx, http.MethodGet, path, nil, result)
}

// AddSession adds a session to a node.
func (c *NodeClient) AddSession(ctx context.Context, id uint64, body, result interface{}) error {
	path, err := c.getURL(ctx, fmt.Sprintf("sessions/%d/keys", id))
	if err != nil {
		return fmt.Errorf("failed to get url: %w", err)
	}

	return c.do(ctx, http.MethodPost, path, body, result)
}
