package types

import (
	"context"
	"time"
)

// ServiceType represents the type of service as a byte.
type ServiceType byte

const (
	ServiceTypeUnspecified ServiceType = 0x00 + iota // ServiceTypeUnspecified represents an unspecified service type.
	ServiceTypeWireGuard                             // ServiceTypeWireGuard represents the WireGuard service type.
	ServiceTypeV2Ray                                 // ServiceTypeV2Ray represents the V2Ray service type.
	ServiceTypeOpenVPN                               // ServiceTypeOpenVPN represents the OpenVPN service type.
)

// String returns the string representation of the ServiceType.
func (s ServiceType) String() string {
	switch s {
	case ServiceTypeWireGuard:
		return "wireguard"
	case ServiceTypeV2Ray:
		return "v2ray"
	case ServiceTypeOpenVPN:
		return "openvpn"
	default:
		return ""
	}
}

// ServiceTypeFromString converts a string to a ServiceType.
func ServiceTypeFromString(s string) ServiceType {
	switch s {
	case "wireguard":
		return ServiceTypeWireGuard
	case "v2ray":
		return ServiceTypeV2Ray
	case "openvpn":
		return ServiceTypeOpenVPN
	default:
		return ServiceTypeUnspecified
	}
}

// PeerStatistics holds network usage metrics for a peer.
type PeerStatistics struct {
	CreatedAt time.Time `json:"created_at,omitempty"` // When this stats record was first created
	UpdatedAt time.Time `json:"updated_at,omitempty"` // When this stats record was last updated

	RxBytes int64 `json:"rx_bytes,omitempty"` // Total uplink bytes received in this snapshot
	TxBytes int64 `json:"tx_bytes,omitempty"` // Total downlink bytes transmitted in this snapshot
}

// NewPeerStatistics creates a new stats record with the given timestamp.
func NewPeerStatistics(t time.Time) *PeerStatistics {
	return &PeerStatistics{
		CreatedAt: t,
		UpdatedAt: t,
	}
}

// Duration returns the elapsed time between creation and last update.
func (s *PeerStatistics) Duration() time.Duration {
	return s.UpdatedAt.Sub(s.CreatedAt)
}

// ClientService defines the interface for client-side service operations.
type ClientService interface {
	Type() ServiceType     // Type returns the type of the client service.
	Init(force bool) error // Init initializes the service, optionally overwriting the config if force is true.

	IsUp() (bool, error) // IsUp checks if the client service is currently running.

	PreUp(req interface{}) error      // PreUp performs operations before the service is brought up.
	Up(ctx context.Context) error     // Up brings up the client service.
	PostUp(ctx context.Context) error // PostUp performs operations after the service is brought up.

	Wait() error // Wait blocks until the client service finishes.

	PreDown() error  // PreDown performs operations before the service is brought down.
	Down() error     // Down brings down the client service.
	PostDown() error // PostDown performs operations after the service is brought down.

	Statistics() (int64, int64, error) // Statistics returns the download and upload statistics.
}

// ServerService defines the interface for server-side service operations.
type ServerService interface {
	Type() ServiceType     // Type returns the type of the server service.
	Init(force bool) error // Init initializes the service, optionally overwriting the config if force is true.

	IsUp() (bool, error) // IsUp checks if the server service is currently running.

	PreUp(req interface{}) error      // PreUp performs operations before the service is brought up.
	Up(ctx context.Context) error     // Up brings up the server service.
	PostUp(ctx context.Context) error // PostUp performs operations after the service is brought up.

	Wait() error // Wait blocks until the server service finishes.

	PreDown() error  // PreDown performs operations before the service is brought down.
	Down() error     // Down brings down the server service.
	PostDown() error // PostDown performs operations after the service is brought down.

	AddPeer(ctx context.Context, req interface{}) (id string, res interface{}, err error) // AddPeer adds a peer and returns its ID, the peer object, and an error if any.
	HasPeer(ctx context.Context, id string) (bool, error)                                 // HasPeer checks if a peer exists in the server service.
	RemovePeer(ctx context.Context, id string) error                                      // RemovePeer removes a peer and returns its ID and error if any.
	PeersLen() int                                                                        // PeersLen returns the number of peers.
	PeerStatistics() (map[string]*PeerStatistics, error)                                  // PeerStatistics returns the statistics for all peers.
}
