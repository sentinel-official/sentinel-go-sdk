package openvpn

import (
	"encoding/hex"

	"github.com/v2fly/v2ray-core/v5/common/uuid"
)

// AddPeerRequest represents a request to add a new peer to the OpenVPN server.
type AddPeerRequest struct {
	UUID uuid.UUID `json:"uuid"` // Unique identifier for the peer
}

// Bytes returns the raw byte representation of the UUID.
func (r *AddPeerRequest) Bytes() []byte {
	return r.UUID.Bytes()
}

// Key returns the hexadecimal-encoded string of the UUID bytes.
func (r *AddPeerRequest) Key() string {
	buf := r.Bytes()
	return hex.EncodeToString(buf)
}

// Validate checks the validity of the AddPeerRequest.
// Currently a no-op; implement as needed.
func (r *AddPeerRequest) Validate() error {
	return nil
}

// NewAddPeerRequestFromBytes constructs an AddPeerRequest from a UUID byte slice.
func NewAddPeerRequestFromBytes(data []byte) (*AddPeerRequest, error) {
	buf, err := uuid.ParseBytes(data)
	if err != nil {
		return nil, err
	}

	return &AddPeerRequest{
		UUID: buf,
	}, nil
}

// NewAddPeerRequestFromKey constructs an AddPeerRequest from a hex-encoded UUID string.
func NewAddPeerRequestFromKey(s string) (*AddPeerRequest, error) {
	data, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}

	return NewAddPeerRequestFromBytes(data)
}

// HasPeerRequest represents a request to check whether a peer exists.
type HasPeerRequest struct {
	UUID uuid.UUID `json:"uuid"` // Unique identifier of the peer to check
}

// Bytes returns the raw byte representation of the UUID.
func (r *HasPeerRequest) Bytes() []byte {
	return r.UUID.Bytes()
}

// Key returns the hexadecimal-encoded string of the UUID bytes.
func (r *HasPeerRequest) Key() string {
	buf := r.Bytes()
	return hex.EncodeToString(buf)
}

// Validate checks the validity of the HasPeerRequest.
// Currently a no-op; implement as needed.
func (r *HasPeerRequest) Validate() error {
	return nil
}

// NewHasPeerRequestFromBytes constructs a HasPeerRequest from a UUID byte slice.
func NewHasPeerRequestFromBytes(data []byte) (*HasPeerRequest, error) {
	buf, err := uuid.ParseBytes(data)
	if err != nil {
		return nil, err
	}

	return &HasPeerRequest{
		UUID: buf,
	}, nil
}

// NewHasPeerRequestFromKey constructs a HasPeerRequest from a hex-encoded UUID string.
func NewHasPeerRequestFromKey(s string) (*HasPeerRequest, error) {
	data, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}

	return NewHasPeerRequestFromBytes(data)
}

// RemovePeerRequest represents a request to remove an existing peer.
type RemovePeerRequest struct {
	UUID uuid.UUID `json:"uuid"` // Unique identifier of the peer to be removed
}

// Bytes returns the raw byte representation of the UUID.
func (r *RemovePeerRequest) Bytes() []byte {
	return r.UUID.Bytes()
}

// Key returns the hexadecimal-encoded string of the UUID bytes.
func (r *RemovePeerRequest) Key() string {
	buf := r.Bytes()
	return hex.EncodeToString(buf)
}

// Validate checks the validity of the RemovePeerRequest.
// Currently a no-op; implement as needed.
func (r *RemovePeerRequest) Validate() error {
	return nil
}

// NewRemovePeerRequestFromBytes constructs a RemovePeerRequest from a UUID byte slice.
func NewRemovePeerRequestFromBytes(data []byte) (*RemovePeerRequest, error) {
	buf, err := uuid.ParseBytes(data)
	if err != nil {
		return nil, err
	}

	return &RemovePeerRequest{
		UUID: buf,
	}, nil
}

// NewRemovePeerRequestFromKey constructs a RemovePeerRequest from a hex-encoded UUID string.
func NewRemovePeerRequestFromKey(s string) (*RemovePeerRequest, error) {
	data, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}

	return NewRemovePeerRequestFromBytes(data)
}
