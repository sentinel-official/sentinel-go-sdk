package hysteria2

// ServerMetadata represents metadata for a Hysteria2 server connection.
type ServerMetadata struct {
	Port         uint16 `json:"port"`          // Port defines the server listening port.
	TLSPin       string `json:"tls_pin"`       // TLSPin is the SHA-256 certificate fingerprint (hex-encoded).
	ObfsPassword string `json:"obfs_password"` // ObfsPassword is the Salamander obfuscation password (empty if disabled).
}
