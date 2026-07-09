package types

import (
	"context"
	"time"

	"github.com/spf13/pflag"
)

// ServiceType represents the type of service as a byte.
type ServiceType byte

const (
	ServiceTypeUnspecified ServiceType = 0x00 + iota // ServiceTypeUnspecified represents an unspecified service type.
	ServiceTypeWireGuard                             // ServiceTypeWireGuard represents the WireGuard service type.
	ServiceTypeV2Ray                                 // ServiceTypeV2Ray represents the V2Ray service type.
	ServiceTypeOpenVPN                               // ServiceTypeOpenVPN represents the OpenVPN service type.
	ServiceTypeXray                                  // ServiceTypeXray represents the Xray service type.
	ServiceTypeAmneziaWG                             // ServiceTypeAmneziaWG represents the AmneziaWG service type.
	ServiceTypeHysteria2                             // ServiceTypeHysteria2 represents the Hysteria2 service type.
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
	case ServiceTypeXray:
		return "xray"
	case ServiceTypeAmneziaWG:
		return "amneziawg"
	case ServiceTypeHysteria2:
		return "hysteria2"
	case ServiceTypeUnspecified:
		return StringUnspecified
	default:
		return ""
	}
}

// ServiceTypeFromString converts a string to a corresponding ServiceType.
// Returns ServiceTypeUnspecified if the string does not match any known service.
func ServiceTypeFromString(s string) ServiceType {
	switch s {
	case "wireguard":
		return ServiceTypeWireGuard
	case "v2ray":
		return ServiceTypeV2Ray
	case "openvpn":
		return ServiceTypeOpenVPN
	case "xray":
		return ServiceTypeXray
	case "amneziawg":
		return ServiceTypeAmneziaWG
	case "hysteria2":
		return ServiceTypeHysteria2
	default:
		return ServiceTypeUnspecified
	}
}

// ServiceConfig defines the interface for service configuration operations.
type ServiceConfig interface {
	Validate() error                              // Validate checks whether the configuration is valid.
	SetForFlags(fs *pflag.FlagSet, prefix string) // SetForFlags binds configuration fields to CLI flags using a prefix.
	ReadAppConfig(file string) error              // ReadAppConfig reads the application config from a file.
	WriteAppConfig(file string) error             // WriteAppConfig writes the application config to a file.
	WriteServiceConfig(file string) error         // WriteServiceConfig writes the service-specific config to a file.
}

// BaseService defines the common behavior shared by all services.
type BaseService interface {
	Type() ServiceType        // Type returns the type of the service.
	IsRunning() (bool, error) // IsRunning checks if the service is currently running.

	Init(force bool) error                                 // Init initializes the service, optionally overwriting the config if force is true.
	Setup(ctx context.Context) error                       // Setup prepares the service for operation.
	Start(parent context.Context) (context.Context, error) // Start brings up the service.
	Stop() error                                           // Stop shuts down the service.
	Wait(ctx context.Context) error                        // Wait blocks until the service finishes.
	Cleanup() error                                        // Cleanup performs cleanup operations after stopping the service.
}

// ClientService defines the interface for client-side service operations.
type ClientService interface {
	BaseService                                           // Embed the common base service
	Statistics(ctx context.Context) (int64, int64, error) // Statistics returns the download and upload statistics.
}

// ServerService defines the interface for server-side service operations.
type ServerService interface {
	BaseService                                                           // Embed the common base service
	Metadata() any                                                        // Metadata returns the service metadata.
	AddPeer(ctx context.Context, req any) (id string, res any, err error) // AddPeer adds a peer and returns its ID, the peer object, and an error if any.
	HasPeer(ctx context.Context, id string) (bool, error)                 // HasPeer checks if a peer exists in the server service.
	RemovePeer(ctx context.Context, id string) error                      // RemovePeer removes a peer by ID and returns an error if any.
	PeersLen() int                                                        // PeersLen returns the number of peers currently configured.
	PeerStatistics() (map[string]*PeerStatistics, error)                  // PeerStatistics returns the statistics for all peers.
}

// PeerStatistics holds network usage metrics for a peer.
type PeerStatistics struct {
	CreatedAt time.Time `json:"created_at,omitzero"` // When this stats record was first created.
	UpdatedAt time.Time `json:"updated_at,omitzero"` // When this stats record was last updated.

	RxBytes int64 `json:"rx_bytes,omitempty"` // Total uplink bytes received in this snapshot.
	TxBytes int64 `json:"tx_bytes,omitempty"` // Total downlink bytes transmitted in this snapshot.
}

// NewPeerStatistics creates a new PeerStatistics instance with a given timestamp.
func NewPeerStatistics(t time.Time) *PeerStatistics {
	return &PeerStatistics{
		CreatedAt: t,
		UpdatedAt: t,
	}
}

// Duration returns the elapsed time between creation and last update of the statistics.
func (s *PeerStatistics) Duration() time.Duration {
	return s.UpdatedAt.Sub(s.CreatedAt)
}
