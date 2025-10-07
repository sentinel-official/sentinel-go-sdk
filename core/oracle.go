package core

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sentinel-official/sentinelhub/v12/x/oracle/types/v1"
)

const (
	// gRPC methods for querying oracle assets information.
	methodQueryAsset  = "/sentinel.oracle.v1.QueryService/QueryAsset"  // Retrieve details of a specific asset
	methodQueryAssets = "/sentinel.oracle.v1.QueryService/QueryAssets" // Retrieve a list of assets
)

// Asset retrieves details of a specific asset by its denom.
// Returns the asset details and any error encountered.
func (c *Client) Asset(ctx context.Context, denom string) (res *v1.Asset, err error) {
	var (
		resp v1.QueryAssetResponse
		req  = &v1.QueryAssetRequest{Denom: denom}
	)

	// Perform the ABCI query to fetch the asset details.
	if err := c.QueryABCI(ctx, methodQueryAsset, req, &resp); err != nil {
		return nil, HandleQueryErr(err)
	}

	return &resp.Asset, nil
}

// Assets retrieves a paginated list of assets.
// Returns the assets, pagination details, and any error encountered.
func (c *Client) Assets(ctx context.Context, pageReq *query.PageRequest) (res []v1.Asset, pageRes *query.PageResponse, err error) {
	var (
		resp v1.QueryAssetsResponse
		req  = &v1.QueryAssetsRequest{
			Pagination: pageReq,
		}
	)

	// Perform the ABCI query to fetch the assets.
	if err := c.QueryABCI(ctx, methodQueryAssets, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	return resp.Assets, resp.Pagination, nil
}
