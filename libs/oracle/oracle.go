package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cosmos/cosmos-sdk/types"
)

// Client defines the common interface for any price oracle client.
// It provides a method to get the quoted price of a base asset.
type Client interface {
	GetQuotePrice(ctx context.Context, basePrice types.DecCoin) (types.Coin, error)
}

// baseClient provides shared HTTP and asset-loading functionality
// used by specific oracle implementations (e.g., CoinGecko, Osmosis).
type baseClient struct {
	*http.Client

	baseURL string           // Base API endpoint (e.g., CoinGecko or Osmosis API).
	m       map[string]Asset // Loaded asset metadata for price lookups.
}

// newBaseClient initializes a baseClient with a given API URL.
// It also loads asset metadata from the embedded assets.json file.
func newBaseClient(url string) *baseClient {
	m, err := LoadAssets()
	if err != nil {
		panic(err)
	}

	return &baseClient{
		Client:  &http.Client{},
		baseURL: url,
		m:       m,
	}
}

// do executes an HTTP request and unmarshals the JSON response into result.
func (c *baseClient) do(ctx context.Context, method string, path string, queries []string, result interface{}) error {
	// Build full request URL with base path and query parameters.
	path, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("constructing URL: %w", err)
	}

	if len(queries) > 0 {
		path = path + "?" + strings.Join(queries, "&")
	}

	// Create HTTP request with context for cancellation/timeouts.
	req, err := http.NewRequestWithContext(ctx, method, path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Send the HTTP request.
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	// Check response status code.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	// Decode JSON body into the provided result object.
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response body: %w", err)
	}

	return nil
}
