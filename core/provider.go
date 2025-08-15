package core

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/sentinel-official/sentinelhub/v12/types"
	"github.com/sentinel-official/sentinelhub/v12/types/v1"
	"github.com/sentinel-official/sentinelhub/v12/x/provider/types/v2"
)

const (
	// gRPC methods for querying provider information
	methodQueryProvider  = "/sentinel.provider.v2.QueryService/QueryProvider"  // Retrieve details of a specific provider
	methodQueryProviders = "/sentinel.provider.v2.QueryService/QueryProviders" // Retrieve a list of providers with optional filtering
)

// Provider retrieves details of a specific provider by its address.
// Returns the provider details and any error encountered.
func (c *Client) Provider(ctx context.Context, provAddr types.ProvAddress) (res *v2.Provider, err error) {
	var (
		resp v2.QueryProviderResponse
		req  = &v2.QueryProviderRequest{Address: provAddr.String()}
	)

	// Perform the gRPC query to fetch the provider details.
	if err := c.QueryGRPC(ctx, methodQueryProvider, req, &resp); err != nil {
		return nil, HandleQueryErr(err)
	}

	return &resp.Provider, nil
}

// Providers retrieves a paginated list of providers filtered by their status.
// Returns the providers, pagination details, and any error encountered.
func (c *Client) Providers(ctx context.Context, status v1.Status, pageReq *query.PageRequest) (res []v2.Provider, pageRes *query.PageResponse, err error) {
	var (
		resp v2.QueryProvidersResponse
		req  = &v2.QueryProvidersRequest{
			Status:     status,
			Pagination: pageReq,
		}
	)

	// Perform the gRPC query to fetch the providers.
	if err := c.QueryGRPC(ctx, methodQueryProviders, req, &resp); err != nil {
		return nil, nil, HandleQueryErr(err)
	}

	return resp.Providers, resp.Pagination, nil
}
