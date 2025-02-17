package node

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/sentinel-official/sentinel-go-sdk/types"
)

// do performs an HTTP request with the given parameters and decodes the response.
func (c *Client) do(ctx context.Context, method, url string, reqBody, result interface{}) error {
	// Create a context with timeout for the HTTP request.
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Configure the HTTP client with TLS settings.
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.insecure,
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

	// Set headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// Perform the HTTP request.
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform request: %w", err)
	}

	defer resp.Body.Close()

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
func (c *Client) getURL(ctx context.Context, pathSuffix string) (string, error) {
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
