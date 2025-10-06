package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"cosmossdk.io/math"
)

// Osmosis represents a client for interacting with the Osmosis API.
type Osmosis struct {
	*http.Client

	apiAddr string // Base URL of the Osmosis API
}

// NewOsmosis creates and returns a new Osmosis client instance.
func NewOsmosis(apiAddr string) *Osmosis {
	return &Osmosis{
		Client:  &http.Client{},
		apiAddr: apiAddr,
	}
}

// ProtoRevPool queries the Osmosis ProtoRev module for the pool ID associated with a given base and quote denomination pair.
func (o *Osmosis) ProtoRevPool(ctx context.Context, baseDenom, quoteDenom string) (uint64, error) {
	path := "/osmosis/protorev/pool"
	queries := []string{
		"base_denom=" + baseDenom,
		"other_denom=" + quoteDenom,
	}

	// Temporary struct to unmarshal the JSON response.
	var r struct {
		PoolID string `json:"pool_id"`
	}

	// Perform the API request.
	if err := o.do(ctx, http.MethodGet, path, queries, &r); err != nil {
		return 0, fmt.Errorf("requesting proto rev pool: %w", err)
	}

	// Convert the pool ID string to uint64.
	poolID, err := strconv.ParseUint(r.PoolID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing pool ID: %w", err)
	}

	return poolID, nil
}

// SpotPrice fetches the current spot price between two denominations using the Osmosis pool determined by ProtoRevPool.
func (o *Osmosis) SpotPrice(ctx context.Context, baseDenom, quoteDenom string) (math.LegacyDec, error) {
	// Get the pool ID for the token pair.
	poolID, err := o.ProtoRevPool(ctx, baseDenom, quoteDenom)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("getting proto rev pool ID: %w", err)
	}

	path := fmt.Sprintf("/osmosis/poolmanager/v2/pools/%d/prices", poolID)
	queries := []string{
		"base_asset_denom=" + baseDenom,
		"quote_asset_denom=" + quoteDenom,
	}

	// Temporary struct to unmarshal the JSON response.
	var r struct {
		SpotPrice string `json:"spot_price"`
	}

	// Perform the API request.
	if err := o.do(ctx, http.MethodGet, path, queries, &r); err != nil {
		return math.LegacyDec{}, fmt.Errorf("requesting spot price: %w", err)
	}

	// Split the spot price string into integer and fractional parts.
	parts := strings.SplitN(r.SpotPrice, ".", 2)

	i, d := parts[0], ""
	if len(parts) == 2 {
		d = parts[1]
	}

	// Limit the decimal precision to 18 places to avoid overflow.
	if len(d) > 18 {
		d = d[:18]
	}

	// Parse the numeric string into a Cosmos SDK decimal type.
	spotPrice, err := math.LegacyNewDecFromStr(i + "." + d)
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("parsing spot price: %w", err)
	}

	return spotPrice, nil
}

// do executes an HTTP request to the Osmosis API and decodes the JSON response into the result.
func (o *Osmosis) do(ctx context.Context, method string, path string, queries []string, result interface{}) error {
	// Construct the full URL path.
	path, err := url.JoinPath(o.apiAddr, path)
	if err != nil {
		return fmt.Errorf("constructing URL: %w", err)
	}

	if len(queries) > 0 {
		path = path + "?" + strings.Join(queries, "&")
	}

	// Create the HTTP request with context.
	req, err := http.NewRequestWithContext(ctx, method, path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Execute the request.
	resp, err := o.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Check for non-OK status codes.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %s", resp.Status)
	}

	// Decode the JSON response into the provided result.
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response body: %w", err)
	}

	return nil
}
