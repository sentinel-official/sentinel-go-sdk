package amneziawg

// ServerMetadata contains metadata for an AmneziaWG server.
//
// In addition to the port and public key (mirroring wireguard/metadata.go),
// it carries the handshake-affecting obfuscation parameters that the client
// must match: S1–S4, H1–H4, and I1–I5. The local-only junk params (Jc/Jmin/Jmax)
// are defaulted on the client side and are not propagated here.
type ServerMetadata struct {
	Port      uint16 `json:"port"`       // Port on which the server listens.
	PublicKey *Key   `json:"public_key"` // Server's public key.

	// Handshake-affecting obfuscation parameters (must match on both ends).
	S1 uint16 `json:"s1"` // S1 is the first handshake obfuscation size.
	S2 uint16 `json:"s2"` // S2 is the second handshake obfuscation size.
	S3 uint16 `json:"s3"` // S3 is the third handshake obfuscation size.
	S4 uint16 `json:"s4"` // S4 is the fourth handshake obfuscation size.

	H1 uint32 `json:"h1"` // H1 is the first magic header.
	H2 uint32 `json:"h2"` // H2 is the second magic header.
	H3 uint32 `json:"h3"` // H3 is the third magic header.
	H4 uint32 `json:"h4"` // H4 is the fourth magic header.

	I1 uint32 `json:"i1"` // I1 is the first imitation/signature packet value.
	I2 uint32 `json:"i2"` // I2 is the second imitation/signature packet value.
	I3 uint32 `json:"i3"` // I3 is the third imitation/signature packet value.
	I4 uint32 `json:"i4"` // I4 is the fourth imitation/signature packet value.
	I5 uint32 `json:"i5"` // I5 is the fifth imitation/signature packet value.
}
