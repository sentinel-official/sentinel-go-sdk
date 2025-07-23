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
	Addr         string          `json:"addr"`          // Bech32-encoded address of the node.
	EgressRate   string          `json:"egress_rate"`   // Node's available upload bandwidth capacity (Bytes per second).
	HandshakeDNS bool            `json:"handshake_dns"` // Indicates if the node supports Handshake (HNS) DNS resolution.
	IngressRate  string          `json:"ingress_rate"`  // Node's available download bandwidth capacity (Bytes per second).
	Location     *geoip.Location `json:"location"`      // Geographical location of the node.
	Moniker      string          `json:"moniker"`       // Human-readable name assigned to the node.
	Peers        int             `json:"peers"`         // Number of connected peers.
	ServiceType  string          `json:"service_type"`  // Node service type (e.g., "V2Ray", "WireGuard", "OpenVPN", etc.).
	Version      *version.Info   `json:"version"`       // Version information of the node software.
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
		return nil, fmt.Errorf("failed to get url: %w", err)
	}

	// Send an HTTP GET request to fetch node details.
	var res GetInfoResult
	if err := c.do(ctx, http.MethodGet, path, nil, &res); err != nil {
		return nil, err
	}

	// Return the retrieved node information.
	return &res, nil
}
