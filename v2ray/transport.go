package v2ray

import (
	"github.com/sentinel-official/sentinel-go-sdk/v2/types"
)

// TransportProtocol is a custom type used to represent different transport protocols.
type TransportProtocol byte

// Constants for TransportProtocol type with automatic incrementation for each transport method.
const (
	TransportProtocolUnspecified  TransportProtocol = iota // Default value for unspecified transport protocol
	TransportProtocolDomainSocket                          // TransportProtocolDomainSocket represents a UNIX domain socket
	TransportProtocolGUN                                   // TransportProtocolGUN represents the GUN protocol
	TransportProtocolGRPC                                  // TransportProtocolGRPC represents gRPC, a high-performance RPC framework
	TransportProtocolHTTP                                  // TransportProtocolHTTP represents the HTTP protocol
	TransportProtocolMKCP                                  // TransportProtocolMKCP represents the MKCP (modified KCP) protocol
	TransportProtocolQUIC                                  // TransportProtocolQUIC represents the QUIC protocol
	TransportProtocolTCP                                   // TransportProtocolTCP represents the TCP transport protocol
	TransportProtocolWebSocket                             // TransportProtocolWebSocket represents the WebSocket protocol
)

// NewTransportProtocolFromString converts a string to a TransportProtocol type.
func NewTransportProtocolFromString(v string) TransportProtocol {
	switch v {
	case "domainsocket":
		return TransportProtocolDomainSocket
	case "gun":
		return TransportProtocolGUN
	case "grpc":
		return TransportProtocolGRPC
	case "http":
		return TransportProtocolHTTP
	case "mkcp":
		return TransportProtocolMKCP
	case "quic":
		return TransportProtocolQUIC
	case "tcp":
		return TransportProtocolTCP
	case "websocket", "ws":
		return TransportProtocolWebSocket
	default:
		return TransportProtocolUnspecified // Returns the default transport protocol type if no match is found
	}
}

// String returns a string representation of the TransportProtocol type.
func (t TransportProtocol) String() string {
	switch t {
	case TransportProtocolDomainSocket:
		return "domainsocket"
	case TransportProtocolGUN:
		return "gun"
	case TransportProtocolGRPC:
		return "grpc"
	case TransportProtocolHTTP:
		return "http"
	case TransportProtocolMKCP:
		return "mkcp"
	case TransportProtocolQUIC:
		return "quic"
	case TransportProtocolTCP:
		return "tcp"
	case TransportProtocolWebSocket:
		return "websocket"
	case TransportProtocolUnspecified:
		return types.StringUnspecified
	default:
		return "" // Return empty string for unknown transport protocol types
	}
}

// IsValid checks if the TransportProtocol value is valid.
func (t TransportProtocol) IsValid() bool {
	return t != TransportProtocolUnspecified && t.String() != ""
}

// TransportSecurity is a custom type used to represent different transport security settings.
type TransportSecurity byte

// Constants for TransportSecurity type with automatic incrementation for each security setting.
const (
	TransportSecurityUnspecified TransportSecurity = iota // Default value for unspecified transport security
	TransportSecurityNone                                 // TransportSecurityNone represents no security
	TransportSecurityTLS                                  // TransportSecurityTLS represents TLS security
)

// NewTransportSecurityFromString converts a string to a TransportSecurity type.
func NewTransportSecurityFromString(v string) TransportSecurity {
	switch v {
	case "none":
		return TransportSecurityNone
	case "tls":
		return TransportSecurityTLS
	default:
		return TransportSecurityUnspecified // Returns the default security if no match is found
	}
}

// String returns a string representation of the TransportSecurity type.
func (t TransportSecurity) String() string {
	switch t {
	case TransportSecurityNone:
		return "none"
	case TransportSecurityTLS:
		return "tls"
	case TransportSecurityUnspecified:
		return types.StringUnspecified
	default:
		return "" // Return empty string for unknown security settings
	}
}

// IsValid checks if the TransportSecurity value is valid.
func (t TransportSecurity) IsValid() bool {
	return t != TransportSecurityUnspecified && t.String() != ""
}
