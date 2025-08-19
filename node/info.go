package node

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sentinel-official/sentinel-go-sdk/libs/geoip"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/version"
)

// GetInfoResult represents metadata about a node.
type GetInfoResult struct {
	Addr         string          `json:"addr"`          // Node address in Bech32 encoding.
	Downlink     string          `json:"downlink"`      // Download capacity from server to client (bytes per second).
	HandshakeDNS bool            `json:"handshake_dns"` // Whether the node supports Handshake (HNS) DNS resolution.
	Location     *geoip.Location `json:"location"`      // Geographical location of the node.
	Moniker      string          `json:"moniker"`       // Human-readable name of the node.
	Peers        int             `json:"peers"`         // Number of connected peers.
	ServiceType  string          `json:"service_type"`  // Node service type (e.g., V2Ray, WireGuard, OpenVPN).
	Uplink       string          `json:"uplink"`        // Upload capacity from client to server (bytes per second).
	Version      *version.Info   `json:"version"`       // Node software version information.
}

// GetServiceType returns the node's service type by converting the ServiceType string into a ServiceType enum.
func (r *GetInfoResult) GetServiceType() types.ServiceType {
	return types.ServiceTypeFromString(r.ServiceType)
}

// GetInfo retrieves detailed information about a specific node.
func (c *Client) GetInfo(ctx context.Context) (*GetInfoResult, error) {
	// Get the API endpoint URL for retrieving node information.
	path, err := c.getURL(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("getting node API URL: %w", err)
	}

	// Send an HTTP GET request to fetch node details.
	var res GetInfoResult
	if err := c.do(ctx, http.MethodGet, path, nil, &res); err != nil {
		return nil, fmt.Errorf("performing get info request: %w", err)
	}

	// Return the retrieved node information.
	return &res, nil
}
