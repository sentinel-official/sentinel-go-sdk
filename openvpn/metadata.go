package openvpn

// ServerMetadata holds metadata about the OpenVPN server configuration.
type ServerMetadata struct {
	Port     uint16 `json:"port"`     // Port number the server listens on
	Protocol string `json:"protocol"` // Transport protocol (e.g., "udp" or "tcp")
	CA       []byte `json:"ca"`       // Certificate Authority (CA) certificate in raw bytes
	TLS      []byte `json:"tls"`      // Static TLS key in raw bytes
}
