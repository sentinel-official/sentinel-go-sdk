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
	DownLink     string          `json:"down_link"`     // Node's available download bandwidth capacity (Bytes per second).
	HandshakeDNS bool            `json:"handshake_dns"` // Indicates if the node supports Handshake (HNS) DNS resolution.
	Location     *geoip.Location `json:"location"`      // Geographical location of the node.
	Moniker      string          `json:"moniker"`       // Human-readable name assigned to the node.
	Peers        int             `json:"peers"`         // Number of connected peers.
	Type         string          `json:"type"`          // Node type (e.g., "V2Ray", "WireGuard", "OpenVPN", etc.).
	UpLink       string          `json:"up_link"`       // Node's available upload bandwidth capacity (Bytes per second).
	Version      *version.Info   `json:"version"`       // Version information of the node software.
}

// GetType returns the node's service type by converting the Type string into a ServiceType enum.
func (r *GetInfoResult) GetType() types.ServiceType {
	return types.ServiceTypeFromString(r.Type)
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
