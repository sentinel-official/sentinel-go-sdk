package oracle

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
)

// Ensure OsmosisClient implements Client interface.
var _ Client = (*OsmosisClient)(nil)

// OsmosisClient represents a client for interacting with the Osmosis API.
type OsmosisClient struct {
	*baseClient
}

// NewOsmosisClient creates and returns a new Osmosis client instance.
func NewOsmosisClient(apiAddr string) *OsmosisClient {
	return &OsmosisClient{
		baseClient: newBaseClient(apiAddr),
	}
}

// ProtoRevPool queries the Osmosis ProtoRev module for the pool ID
// associated with a given base and quote denomination pair.
func (c *OsmosisClient) ProtoRevPool(ctx context.Context, baseDenom, quoteDenom string) (uint64, error) {
	path := "/osmosis/protorev/pool"
	queries := []string{
		"base_denom=" + baseDenom,
		"other_denom=" + quoteDenom,
	}

	// Temporary struct to unmarshal the JSON response.
	var body struct {
		PoolID string `json:"pool_id"`
	}

	// Perform the API request.
	if err := c.Get(ctx, path, queries, &body); err != nil {
		return 0, fmt.Errorf("requesting proto rev pool: %w", err)
	}

	// Convert the pool ID string to uint64.
	poolID, err := strconv.ParseUint(body.PoolID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing pool ID: %w", err)
	}

	return poolID, nil
}

// SpotPrice fetches the current spot price between two denominations
// using the Osmosis pool determined by ProtoRevPool.
func (c *OsmosisClient) SpotPrice(ctx context.Context, poolID uint64, baseDenom, quoteDenom string) (math.LegacyDec, error) {
	path := fmt.Sprintf("/osmosis/poolmanager/v2/pools/%d/prices", poolID)
	queries := []string{
		"base_asset_denom=" + baseDenom,
		"quote_asset_denom=" + quoteDenom,
	}

	// Temporary struct to unmarshal the JSON response.
	var body struct {
		SpotPrice string `json:"spot_price"`
	}

	// Perform the API request.
	if err := c.Get(ctx, path, queries, &body); err != nil {
		return math.LegacyDec{}, fmt.Errorf("requesting spot price: %w", err)
	}

	// Split the spot price string into integer and fractional parts.
	parts := strings.SplitN(body.SpotPrice, ".", 2)

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

// GetQuotePrice calculates the quote price of a given base asset using data from Osmosis pools.
func (c *OsmosisClient) GetQuotePrice(ctx context.Context, basePrice types.DecCoin) (types.Coin, error) {
	// Look up the asset configuration for the given denom.
	asset, ok := c.m[basePrice.Denom]
	if !ok {
		return types.Coin{}, fmt.Errorf("asset for deonm %q does not exist", basePrice.Denom)
	}

	// Get the ProtoRev pool ID for this asset.
	poolID, err := c.ProtoRevPool(ctx, asset.ProtoRevPoolRequest.BaseDenom, asset.ProtoRevPoolRequest.OtherDenom)
	if err != nil {
		return types.Coin{}, fmt.Errorf("getting protorev pool ID: %w", err)
	}

	// Fetch the spot price from Osmosis.
	spotPrice, err := c.SpotPrice(ctx, poolID, asset.SpotPriceRequest.BaseAssetDenom, asset.SpotPriceRequest.QuoteAssetDenom)
	if err != nil {
		return types.Coin{}, fmt.Errorf("getting spot price: %w", err)
	}

	// Adjust for multiplier and compute final quote amount.
	asset.SpotPrice = spotPrice.MulInt(asset.Multiplier())
	amount := basePrice.Amount.Mul(asset.SpotPrice).TruncateInt()

	return types.Coin{Denom: basePrice.Denom, Amount: amount}, nil
}
