package wireguard

import (
	"net/netip"
)

// AddPeerResponse represents the response for adding a peer to the WireGuard server.
type AddPeerResponse struct {
	Addrs    []netip.Prefix    `json:"addrs"`    // Assigned addrs for the peer.
	Metadata []*ServerMetadata `json:"metadata"` // Metadata about the server.
}

// GetAddrs returns the assigned IP addresses as strings.
func (r *AddPeerResponse) GetAddrs() []string {
	var addrs []string

	// Convert netip.Prefix to string.
	for _, addr := range r.Addrs {
		addrs = append(addrs, addr.String())
	}

	return addrs
}
