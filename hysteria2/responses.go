package hysteria2

// AddPeerResponse represents the response returned after adding a peer to the Hysteria2 server.
type AddPeerResponse struct {
	Metadata []*ServerMetadata `json:"metadata"` // Metadata contains the server's connection details.
}
