package oracle

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/sentinel-official/sentinelhub/v12/x/oracle/types/v1"
)

// Querier defines the interface required for fetching asset metadata.
type Querier interface {
	Asset(ctx context.Context, denom string) (*v1.Asset, error)
}

// Client defines the interface for querying spot or quote prices for a given base asset.
type Client interface {
	GetQuotePrice(ctx context.Context, basePrice types.DecCoin) (types.Coin, error)
}
