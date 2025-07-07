package types

import (
	"context"
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

// PeerStatistic represents the download and upload statistics for a peer.
type PeerStatistic struct {
	Key           string `json:"key"`            // Key is the identifier for the peer.
	DownloadBytes int64  `json:"download_bytes"` // DownloadBytes is the total download in bytes.
	UploadBytes   int64  `json:"upload_bytes"`   // UploadBytes is the total upload in bytes.
}

// ClientService defines the interface for client-side service operations.
type ClientService interface {
	Type() ServiceType // Type returns the type of the client service.

	IsUp(context.Context) (bool, error) // IsUp checks if the client service is up.
	PreUp(interface{}) error            // PreUp performs operations before the service is brought up.
	Up(context.Context) error           // Up brings up the client service.
	PostUp() error                      // PostUp performs operations after the service is brought up.

	PreDown() error             // PreDown performs operations before the service is brought down.
	Down(context.Context) error // Down brings down the client service.
	PostDown() error            // PostDown performs operations after the service is brought down.

	Statistics(context.Context) (int64, int64, error) // Statistics returns the download and upload statistics.
}

// ServerService defines the interface for server-side service operations.
type ServerService interface {
	Type() ServiceType // Type returns the type of the server service.

	IsUp(context.Context) (bool, error) // IsUp checks if the server service is up.
	PreUp(interface{}) error            // PreUp performs operations before the service is brought up.
	Up(context.Context) error           // Up brings up the server service.
	PostUp() error                      // PostUp performs operations after the service is brought up.

	PreDown() error             // PreDown performs operations before the service is brought down.
	Down(context.Context) error // Down brings down the server service.
	PostDown() error            // PostDown performs operations after the service is brought down.

	AddPeer(context.Context, interface{}) (interface{}, error) // AddPeer adds a peer to the server service.
	HasPeer(context.Context, interface{}) (bool, error)        // HasPeer checks if a peer exists in the server service.
	RemovePeer(context.Context, interface{}) error             // RemovePeer removes a peer from the server service.
	PeerCount() int                                            // PeerCount returns the count of peers.
	PeerStatistics(context.Context) ([]*PeerStatistic, error)  // PeerStatistics returns the statistics for all peers.
}
