package xray

import (
	"github.com/sentinel-official/sentinel-go-sdk/v2/types"
)

// stringNone is the string representation shared by the none transport security and flow.
const stringNone = "none"

// TransportProtocol is a custom type used to represent different transport protocols.
type TransportProtocol byte

// Constants for TransportProtocol type with automatic incrementation for each transport method.
const (
	TransportProtocolUnspecified TransportProtocol = iota // Default value for unspecified transport protocol
	TransportProtocolTCP                                  // TransportProtocolTCP represents the TCP transport protocol
	TransportProtocolWebSocket                            // TransportProtocolWebSocket represents the WebSocket protocol
	TransportProtocolGRPC                                 // TransportProtocolGRPC represents gRPC, a high-performance RPC framework
	TransportProtocolHTTPUpgrade                          // TransportProtocolHTTPUpgrade represents the HTTPUpgrade protocol
	TransportProtocolXHTTP                                // TransportProtocolXHTTP represents the XHTTP protocol
)

// NewTransportProtocolFromString converts a string to a TransportProtocol type.
func NewTransportProtocolFromString(v string) TransportProtocol {
	switch v {
	case "tcp":
		return TransportProtocolTCP
	case "websocket", "ws":
		return TransportProtocolWebSocket
	case "grpc":
		return TransportProtocolGRPC
	case "httpupgrade":
		return TransportProtocolHTTPUpgrade
	case "xhttp":
		return TransportProtocolXHTTP
	default:
		return TransportProtocolUnspecified // Returns the default transport protocol type if no match is found
	}
}

// String returns a string representation of the TransportProtocol type.
func (t TransportProtocol) String() string {
	switch t {
	case TransportProtocolTCP:
		return "tcp"
	case TransportProtocolWebSocket:
		return "websocket"
	case TransportProtocolGRPC:
		return "grpc"
	case TransportProtocolHTTPUpgrade:
		return "httpupgrade"
	case TransportProtocolXHTTP:
		return "xhttp"
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
	TransportSecurityReality                              // TransportSecurityReality represents Reality security
)

// NewTransportSecurityFromString converts a string to a TransportSecurity type.
func NewTransportSecurityFromString(v string) TransportSecurity {
	switch v {
	case stringNone:
		return TransportSecurityNone
	case "tls":
		return TransportSecurityTLS
	case "reality":
		return TransportSecurityReality
	default:
		return TransportSecurityUnspecified // Returns the default security if no match is found
	}
}

// String returns a string representation of the TransportSecurity type.
func (t TransportSecurity) String() string {
	switch t {
	case TransportSecurityNone:
		return stringNone
	case TransportSecurityTLS:
		return "tls"
	case TransportSecurityReality:
		return "reality"
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

// Flow is a custom type used to represent the VLESS flow control setting.
type Flow byte

// Constants for Flow type with automatic incrementation for each flow setting.
const (
	FlowUnspecified Flow = iota // Default value for unspecified flow
	FlowNone                    // FlowNone represents no flow control
	FlowVision                  // FlowVision represents the xtls-rprx-vision flow control
)

// NewFlowFromString converts a string to a Flow type.
func NewFlowFromString(v string) Flow {
	switch v {
	case stringNone, "":
		return FlowNone
	case "xtls-rprx-vision":
		return FlowVision
	default:
		return FlowUnspecified // Returns the default flow if no match is found
	}
}

// String returns a string representation of the Flow type.
func (f Flow) String() string {
	switch f {
	case FlowNone:
		return stringNone
	case FlowVision:
		return "xtls-rprx-vision"
	case FlowUnspecified:
		return types.StringUnspecified
	default:
		return "" // Return empty string for unknown flow settings
	}
}

// IsValid checks if the Flow value is valid.
func (f Flow) IsValid() bool {
	return f != FlowUnspecified && f.String() != ""
}
