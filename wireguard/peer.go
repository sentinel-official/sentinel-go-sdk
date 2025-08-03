package wireguard

import (
	"net/netip"
	"time"

	"github.com/sentinel-official/sentinel-go-sdk/types"
)

// Peer represents a WireGuard peer with identifying information and network statistics.
type Peer struct {
	ID        string                `json:"id,omitempty"`        // Unique identifier of the peer.
	Addrs     []netip.Addr          `json:"addrs,omitempty"`     // IP addresses assigned to the peer.
	Current   *types.PeerStatistics `json:"current,omitempty"`   // Current session statistics (e.g., active counters).
	Previous  *types.PeerStatistics `json:"previous,omitempty"`  // Accumulated statistics from previous sessions.
	Timestamp time.Time             `json:"timestamp,omitempty"` // Timestamp of when the peer was added or last updated.
}

// TotalDuration returns the total connection duration by summing previous and current durations.
func (p *Peer) TotalDuration() time.Duration {
	return p.Previous.Duration + p.Current.Duration
}

// TotalRxBytes returns the total received bytes by summing previous and current counters.
func (p *Peer) TotalRxBytes() int64 {
	return p.Previous.RxBytes + p.Current.RxBytes
}

// TotalTxBytes returns the total transmitted bytes by summing previous and current counters.
func (p *Peer) TotalTxBytes() int64 {
	return p.Previous.TxBytes + p.Current.TxBytes
}
