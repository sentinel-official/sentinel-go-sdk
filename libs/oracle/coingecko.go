package oracle

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
)

// Ensure CoinGeckoClient implements the Client interface at compile time.
var _ Client = (*CoinGeckoClient)(nil)

// CoinGeckoClient is a lightweight client for interacting with the CoinGeckoClient API.
type CoinGeckoClient struct {
	*baseClient

	apiKey string // Optional API key for authenticated or higher-rate usage.
}

// NewCoinGeckoClient creates a new CoinGeckoClient client using the default API endpoint
// and an optional API key.
func NewCoinGeckoClient(apiKey string) *CoinGeckoClient {
	return &CoinGeckoClient{
		baseClient: newBaseClient("https://api.coingecko.com/api/v3"),
		apiKey:     apiKey,
	}
}

// SpotPrice retrieves the spot price for a given asset ID and target currency.
func (c *CoinGeckoClient) SpotPrice(ctx context.Context, id, currency string) (math.LegacyDec, error) {
	path := "simple/price"
	queries := []string{
		"ids=" + id,
		"vs_currencies=" + currency,
	}

	// Temporary map for JSON decoding.
	var body map[string]map[string]float64

	// Send request to CoinGecko API.
	if err := c.Get(ctx, path, queries, &body); err != nil {
		return math.LegacyDec{}, fmt.Errorf("requesting spot price: %w", err)
	}

	// Extract the nested price value.
	v, ok := body[id][currency]
	if !ok {
		return math.LegacyDec{}, errors.New("spot price does not exist")
	}

	// Convert float to Cosmos SDK decimal.
	spotPrice, err := math.LegacyNewDecFromStr(fmt.Sprintf("%f", v))
	if err != nil {
		return math.LegacyDec{}, fmt.Errorf("parsing spot price: %w", err)
	}

	return spotPrice, nil
}

// GetQuotePrice returns the converted quote value of a base price coin using CoinGecko market data.
func (c *CoinGeckoClient) GetQuotePrice(ctx context.Context, basePrice types.DecCoin) (types.Coin, error) {
	// Lookup asset metadata from preloaded asset map.
	asset, ok := c.m[basePrice.Denom]
	if !ok {
		return types.Coin{}, fmt.Errorf("asset for denom %q does not exist", basePrice.Denom)
	}

	// Get USD spot price from CoinGecko.
	spotPrice, err := c.SpotPrice(ctx, asset.CoinGeckoID, "usd")
	if err != nil {
		return types.Coin{}, fmt.Errorf("getting spot price: %w", err)
	}

	// Compute adjusted quote using the asset multiplier.
	asset.SpotPrice = math.LegacyOneDec().Quo(spotPrice).MulInt(asset.Multiplier())
	amount := basePrice.Amount.Mul(asset.SpotPrice).TruncateInt()

	return types.Coin{Denom: basePrice.Denom, Amount: amount}, nil
}
