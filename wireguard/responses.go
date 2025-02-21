package wireguard

import (
	"net/netip"
)

// AddPeerResponse represents the response for adding a peer to the WireGuard server.
type AddPeerResponse struct {
	Addrs    []netip.Prefix    `json:"addrs"`    // Assigned addrs for the peer.
	Metadata []*ServerMetadata `json:"metadata"` // Metadata about the server.
}
