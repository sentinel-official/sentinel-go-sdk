package oracle

import (
	"context"
	"net/http"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
)

// Ensure CoinGecko implements Client interface.
var _ Client = (*CoinGecko)(nil)

type CoinGecko struct {
	*http.Client

	apiKey string
}

func NewCoinGecko(apiKey string) *CoinGecko {
	return &CoinGecko{
		Client: &http.Client{},
		apiKey: apiKey,
	}
}

func (c *CoinGecko) GetQuotePrice(_ context.Context, basePrice types.DecCoin) (types.Coin, error) {
	return types.Coin{Denom: basePrice.Denom, Amount: math.ZeroInt()}, nil
}
