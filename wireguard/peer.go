package wireguard

import (
	"net/netip"
	"time"
)

// Peer represents a network peer with identity and IP addresses.
type Peer struct {
	ID       string        `json:"id,omitempty"`
	Addrs    []netip.Addr  `json:"addrs,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	RxBytes  int64         `json:"rx_bytes,omitempty"`
	TxBytes  int64         `json:"tx_bytes,omitempty"`
}
