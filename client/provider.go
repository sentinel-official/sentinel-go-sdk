package client

import (
	"context"

	sentinelhub "github.com/sentinel-official/hub/types"
	providertypes "github.com/sentinel-official/hub/x/provider/types"

	"github.com/sentinel-official/sentinel-go-sdk/v1/client/options"
)

// Provider queries and returns information about a specific provider based on the provided provider address.
// It uses gRPC to send a request to the "/sentinel.provider.v2.QueryService/QueryProvider" endpoint.
// The result is a pointer to providertypes.Provider and an error if the query fails.
func (c *Context) Provider(ctx context.Context, provAddr sentinelhub.ProvAddress, opts *options.QueryOptions) (res *providertypes.Provider, err error) {
	// Initialize variables for the query.
	var (
		resp   providertypes.QueryProviderResponse
		method = "/sentinel.provider.v2.QueryService/QueryProvider"
		req    = &providertypes.QueryProviderRequest{
			Address: provAddr.String(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return a pointer to the provider and a nil error.
	return &resp.Provider, nil
}

// Providers queries and returns a list of providers based on the provided status and options.
// It uses gRPC to send a request to the "/sentinel.provider.v2.QueryService/QueryProviders" endpoint.
// The result is a slice of providertypes.Provider and an error if the query fails.
func (c *Context) Providers(ctx context.Context, status sentinelhub.Status, opts *options.QueryOptions) (res []providertypes.Provider, err error) {
	// Initialize variables for the query.
	var (
		resp   providertypes.QueryProvidersResponse
		method = "/sentinel.provider.v2.QueryService/QueryProviders"
		req    = &providertypes.QueryProvidersRequest{
			Status:     status,
			Pagination: opts.PageRequest(),
		}
	)

	// Send a gRPC query using the provided context, method, request, response, and options.
	if err := c.QueryGRPC(ctx, method, req, &resp, opts); err != nil {
		return nil, err
	}

	// Return the list of providers and a nil error.
	return resp.Providers, nil
}
