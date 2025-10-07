package oracle

import (
	"context"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
)

// Ensure CoinGecko implements Client interface.
var _ Client = (*CoinGecko)(nil)

type CoinGecko struct{}

func (c *CoinGecko) GetQuotePrice(_ context.Context, basePrice types.DecCoin) (types.Coin, error) {
	return types.Coin{Denom: basePrice.Denom, Amount: math.ZeroInt()}, nil
}
