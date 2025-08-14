package v2ray

import (
	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
)

// ServerMetadata represents metadata for a V2Ray server's inbound connection.
type ServerMetadata struct {
	Port              string            `json:"port"`               // Port defines the outbound port range.
	ProxyProtocol     ProxyProtocol     `json:"proxy_protocol"`     // ProxyProtocol specifies the proxy protocol type.
	TransportProtocol TransportProtocol `json:"transport_protocol"` // TransportProtocol specifies the transport protocol type.
	TransportSecurity TransportSecurity `json:"transport_security"` // TransportSecurity specifies the transport security type.
}

// GetPort parses the Port field and returns it as a netip.Port.
// It panics if the port string is invalid.
func (m *ServerMetadata) GetPort() *netip.Port {
	port, err := netip.NewPortFromString(m.Port)
	if err != nil {
		panic(err)
	}

	return port
}
