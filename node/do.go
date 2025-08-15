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
			return fmt.Errorf("marshalling request body: %w", err)
		}

		body = bytes.NewReader(buf)
	}

	// Create the HTTP request.
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating %q request to %q: %w", method, url, err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	// Perform the HTTP request.
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("performing %q request to %q: %w", method, url, err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Decode the JSON response into a predefined structure.
	var respBody types.Response
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return fmt.Errorf("unmarshalling response body: %w", err)
	}

	// Check for errors in the response.
	if err := respBody.Err(); err != nil {
		return fmt.Errorf("response body contains error: %w", err)
	}

	// Decode the Result field if a result target is provided.
	if result != nil {
		buf, err := json.Marshal(respBody.Result)
		if err != nil {
			return fmt.Errorf("marshalling response body result: %w", err)
		}
		if err := json.Unmarshal(buf, result); err != nil {
			return fmt.Errorf("unmarshalling response body result: %w", err)
		}
	}

	return nil
}

// getURL constructs the full URL for a node with an optional path.
func (c *Client) getURL(ctx context.Context, pathSuffix string) (string, error) {
	node, err := c.Node(ctx, c.addr)
	if err != nil {
		return "", fmt.Errorf("querying node %q: %w", c.addr, err)
	}
	if node == nil {
		return "", fmt.Errorf("node %q does not exist", c.addr)
	}

	// Construct base URL with HTTPS scheme.
	addr := "https" + "://" + node.RemoteAddrs[0]

	// Join base URL with the provided path suffix.
	path, err := url.JoinPath(addr, pathSuffix)
	if err != nil {
		return "", fmt.Errorf("constructing URL path %q: %w", pathSuffix, err)
	}

	return path, nil
}
