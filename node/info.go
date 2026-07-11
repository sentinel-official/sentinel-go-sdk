package node

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sentinel-official/sentinel-go-sdk/v2/libs/geoip"
	"github.com/sentinel-official/sentinel-go-sdk/v2/version"
)

// ServiceInfo describes one protocol a node serves.
type ServiceInfo struct {
	Metadata any    `json:"metadata"` // Protocol-specific service metadata.
	Peers    int    `json:"peers"`    // Number of connections for this service.
	Type     string `json:"type"`     // Service type (e.g., "wireguard", "amneziawg", "hysteria2").
}

// GetInfoResult represents metadata about a node.
type GetInfoResult struct {
	Addr         string          `json:"addr"`          // Node address in Bech32 encoding.
	Downlink     string          `json:"downlink"`      // Download capacity from server to client (bytes per second).
	HandshakeDNS bool            `json:"handshake_dns"` // Whether the node supports Handshake (HNS) DNS resolution.
	Location     *geoip.Location `json:"location"`      // Geographical location of the node.
	Moniker      string          `json:"moniker"`       // Human-readable name of the node.
	Peers        int             `json:"peers"`         // Total number of connected peers across services.
	Services     []ServiceInfo   `json:"services"`      // Per-protocol service info; one entry per active service.
	Uplink       string          `json:"uplink"`        // Upload capacity from client to server (bytes per second).
	Version      *version.Info   `json:"version"`       // Node software version information.
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
