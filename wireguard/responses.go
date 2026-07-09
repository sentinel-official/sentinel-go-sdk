package wireguard

import (
	"github.com/sentinel-official/sentinel-go-sdk/v2/libs/netip"
)

// AddPeerResponse represents the response for adding a peer to the WireGuard server.
type AddPeerResponse struct {
	Addrs    []*netip.Prefix   `json:"addrs"`    // Assigned addrs for the peer.
	Metadata []*ServerMetadata `json:"metadata"` // Metadata about the server.
}

// GetAddrs returns the assigned IP addresses as strings.
func (r *AddPeerResponse) GetAddrs() []string {
	addrs := make([]string, 0, len(r.Addrs))

	// Convert netip.Prefix to string.
	for _, addr := range r.Addrs {
		addrs = append(addrs, addr.String())
	}

	return addrs
}
