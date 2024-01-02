package types

// Transport represents different transport protocols supported by the system.
type Transport byte

const (
	// TransportUnspecified represents an unspecified or unknown transport protocol.
	TransportUnspecified Transport = 0x00 + iota
	// TransportTCP represents the TCP transport protocol.
	TransportTCP
	// TransportMKCP represents the MKCP transport protocol.
	TransportMKCP
	// TransportWebSocket represents the WebSocket transport protocol.
	TransportWebSocket
	// TransportHTTP represents the HTTP transport protocol.
	TransportHTTP
	// TransportDomainSocket represents the Domain Socket transport protocol.
	TransportDomainSocket
	// TransportQUIC represents the QUIC transport protocol.
	TransportQUIC
	// TransportGUN represents the GUN transport protocol.
	TransportGUN
	// TransportGRPC represents the gRPC transport protocol.
	TransportGRPC
)

// String returns a human-readable string representation of the Transport type.
func (t Transport) String() string {
	switch t {
	case TransportTCP:
		return "tcp"
	case TransportMKCP:
		return "mkcp"
	case TransportWebSocket:
		return "websocket"
	case TransportHTTP:
		return "http"
	case TransportDomainSocket:
		return "domainsocket"
	case TransportQUIC:
		return "quic"
	case TransportGUN:
		return "gun"
	case TransportGRPC:
		return "grpc"
	default:
		return ""
	}
}

// NewTransportFromString converts a string representation to the corresponding Transport type.
func NewTransportFromString(v string) Transport {
	switch v {
	case "tcp":
		return TransportTCP
	case "mkcp":
		return TransportMKCP
	case "websocket":
		return TransportWebSocket
	case "http":
		return TransportHTTP
	case "domainsocket":
		return TransportDomainSocket
	case "quic":
		return TransportQUIC
	case "gun":
		return TransportGUN
	case "grpc":
		return TransportGRPC
	default:
		return TransportUnspecified
	}
}
