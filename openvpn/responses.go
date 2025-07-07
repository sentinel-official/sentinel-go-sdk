package openvpn

// AddPeerResponse represents the response returned after adding a peer to the OpenVPN server.
type AddPeerResponse struct {
	Metadata []*ServerMetadata `json:"metadata"` // Server's connection metadata (port, protocol, cert info)
	Cert     []byte            `json:"cert"`     // Client certificate in DER format
	Key      []byte            `json:"key"`      // Client private key in DER format
}
