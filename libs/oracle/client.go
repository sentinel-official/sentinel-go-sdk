package oracle

import (
	"context"

	"github.com/sentinel-official/sentinelhub/v12/x/oracle/types/v1"
)

// ClientKey is a context key used to store and retrieve a Client instance from a context.Context value.
type ClientKey struct{}

// Client defines the interface required by the Osmosis oracle integration to fetch asset metadata.
type Client interface {
	Asset(ctx context.Context, denom string) (*v1.Asset, error)
}
