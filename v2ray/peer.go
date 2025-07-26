package v2ray

import (
	"time"
)

// Peer represents an entity with an Email field.
type Peer struct {
	ID       string        `json:"id,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	RxBytes  int64         `json:"rx_bytes,omitempty"`
	TxBytes  int64         `json:"tx_bytes,omitempty"`
}
