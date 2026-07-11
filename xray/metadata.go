package xray

import (
	"github.com/sentinel-official/sentinel-go-sdk/v2/libs/netip"
)

// ServerMetadata represents metadata for an Xray server's inbound connection.
type ServerMetadata struct {
	Port              string            `json:"port"`               // Port defines the outbound port range.
	ProxyProtocol     ProxyProtocol     `json:"proxy_protocol"`     // ProxyProtocol specifies the proxy protocol type.
	TransportProtocol TransportProtocol `json:"transport_protocol"` // TransportProtocol specifies the transport protocol type.
	TransportSecurity TransportSecurity `json:"transport_security"` // TransportSecurity specifies the transport security type.
	Flow              Flow              `json:"flow"`               // Flow specifies the VLESS flow control setting.
	Method            string            `json:"method"`             // Method specifies the Shadowsocks 2022 method.
	Key               string            `json:"key"`                // Key specifies the Shadowsocks 2022 server-level key (iPSK).
	TLSPin            string            `json:"tls_pin"`            // TLSPin specifies the SHA-256 pin of the server's TLS certificate.

	RealityServerName  string `json:"reality_server_name"` // RealityServerName specifies the Reality SNI value.
	RealityShortId     string `json:"reality_short_id"`    // RealityShortId specifies the Reality shortId value.
	RealityPublicKey   string `json:"reality_public_key"`  // RealityPublicKey specifies the Reality public key.
	RealityFingerprint string `json:"reality_fingerprint"` // RealityFingerprint specifies the Reality uTLS fingerprint.
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
