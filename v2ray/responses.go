package v2ray

// AddPeerResponse represents the response returned after adding a peer to the V2Ray server.
type AddPeerResponse struct {
	Metadata []*ServerMetadata `json:"metadata"` // Metadata contains the server's inbound connection details.
}
