package oracle

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/sentinel-official/sentinelhub/v12/x/oracle/types/v1"
)

// AssetQuerierKey is a context key used to store and retrieve an AssetQuerier
// instance from a context.Context value.
type AssetQuerierKey struct{}

// AssetQuerier defines the interface required by the Osmosis oracle integration to fetch asset metadata.
type AssetQuerier interface {
	Asset(ctx context.Context, denom string) (*v1.Asset, error)
}

// Client defines the interface for querying spot or quote prices for a given base asset.
type Client interface {
	GetQuotePrice(ctx context.Context, basePrice types.DecCoin) (types.Coin, error)
}
