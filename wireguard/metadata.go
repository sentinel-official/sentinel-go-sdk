package wireguard

// ServerMetadata contains metadata for a WireGuard server.
type ServerMetadata struct {
	Port      uint16 `json:"port"`       // Port on which the server listens.
	PublicKey *Key   `json:"public_key"` // Server's public key.
}
