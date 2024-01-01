package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	Download sdk.Int
	Key      string
	Upload   sdk.Int
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
	Statistics() (sdk.Int, sdk.Int, error)
	Up() error
}

// ServerService defines the interface for server-side network services.
type ServerService interface {
	AddPeer(data []byte) ([]byte, error)
	HasPeer(data []byte) bool
	Info() []byte
	Init(homeDir string) error
	PeerCount() int
	PeerStatistics() ([]*PeerStatistic, error)
	RemovePeer(data []byte) error
	Start() error
	Stop() error
	Type() ServiceType
}
