package types

import (
	"context"
)

// ServiceType represents different types of network services supported by the system.
type ServiceType byte

const (
	// ServiceTypeUnspecified represents an unspecified or unknown service type.
	ServiceTypeUnspecified ServiceType = 0x00 + iota
	// ServiceTypeWireGuard represents the WireGuard service type.
	ServiceTypeWireGuard
	// ServiceTypeV2Ray represents the V2Ray service type.
	ServiceTypeV2Ray
)

// String returns a human-readable string representation of the ServiceType.
func (s ServiceType) String() string {
	switch s {
	case ServiceTypeWireGuard:
		return "wireguard"
	case ServiceTypeV2Ray:
		return "v2ray"
	default:
		return ""
	}
}

// ServiceTypeFromString converts a string representation to the corresponding ServiceType.
func ServiceTypeFromString(s string) ServiceType {
	switch s {
	case "wireguard":
		return ServiceTypeWireGuard
	case "v2ray":
		return ServiceTypeV2Ray
	default:
		return ServiceTypeUnspecified
	}
}

// PeerStatistic represents statistics for a peer's network activity.
type PeerStatistic struct {
	Download int64  `json:"download"`
	Key      string `json:"key"`
	Upload   int64  `json:"upload"`
}

// ClientService defines the interface for client-side network services.
type ClientService interface {
	Down() error
	Info() []byte
	IsUp() bool
	PostDown() error
	PostUp() error
	PreDown() error
	PreUp() error
	Statistics() (int64, int64, error)
	Up() error
}

// ServerService defines the interface for server-side network services.
type ServerService interface {
	AddPeer(context.Context, []byte) ([]byte, error)
	HasPeer(context.Context, []byte) (bool, error)
	Info() []byte
	Init() error
	PeerCount() int
	PeerStatistics(context.Context) ([]*PeerStatistic, error)
	RemovePeer(context.Context, []byte) error
	Start() error
	Stop() error
	Type() ServiceType
}
