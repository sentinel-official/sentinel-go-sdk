package node

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// PeerRequest wraps one protocol-specific peer request with a self-describing service type.
type PeerRequest struct {
	Data []byte `binding:"required,gt=0" json:"data"` // JSON-marshaled protocol-specific peer request, required and non-empty.
	Type string `binding:"required,gt=0" json:"type"` // Service type, required and non-empty.
}

// AddPeerResponse wraps one protocol-specific add-peer response with a self-describing service type.
type AddPeerResponse struct {
	Data []byte `json:"data"` // JSON-marshaled protocol-specific response on success; empty on failure.
	Err  string `json:"err"`  // Non-empty error string on failure; empty on success.
	Type string `json:"type"` // Service type matching the corresponding PeerRequest.
}

// InitHandshakeRequestBody represents the request payload for initiating a multi-protocol handshake.
type InitHandshakeRequestBody struct {
	ID           uint64        `binding:"required,gt=0"        json:"id"`            // Unique session identifier, must be greater than zero.
	PeerRequests []PeerRequest `binding:"required,gt=0,dive"   json:"peer_requests"` // Per-protocol peer requests, non-empty; each element validated via PeerRequest binding tags.
	PubKey       string        `binding:"required,gt=0"        json:"pub_key"`       // Public key associated with the session, required and non-empty.
	Signature    string        `binding:"required,base64,gt=0" json:"signature"`     // Digital signature in Base64, required and non-empty.
}

// AccAddr converts the public key into a Cosmos SDK AccAddress.
func (r *InitHandshakeRequestBody) AccAddr() (types.AccAddress, error) {
	// Decode the public key.
	pubKey, err := utils.DecodePubKey(r.PubKey)
	if err != nil {
		return nil, fmt.Errorf("decoding public key %q: %w", r.PubKey, err)
	}

	return pubKey.Address().Bytes(), nil
}

// Msg constructs the message for signing: BigEndian(ID) followed by each peer
// request's Type bytes then Data bytes, in request order (no length prefix).
func (r *InitHandshakeRequestBody) Msg() (buf []byte) {
	buf = append(buf, types.Uint64ToBigEndian(r.ID)...)
	for i := range r.PeerRequests {
		buf = append(buf, []byte(r.PeerRequests[i].Type)...)
		buf = append(buf, r.PeerRequests[i].Data...)
	}

	return buf
}

// Verify checks whether the provided signature is valid for the given message and public key.
func (r *InitHandshakeRequestBody) Verify() error {
	// Decode the public key.
	pubKey, err := utils.DecodePubKey(r.PubKey)
	if err != nil {
		return fmt.Errorf("decoding public key %q: %w", r.PubKey, err)
	}

	// Decode the signature from Base64.
	signature, err := base64.StdEncoding.DecodeString(r.Signature)
	if err != nil {
		return fmt.Errorf("decoding signature %q: %w", r.Signature, err)
	}

	// Verify the signature against the message and public key.
	if !pubKey.VerifySignature(r.Msg(), signature) {
		return fmt.Errorf("signature verification failed for session %d", r.ID)
	}

	return nil
}

// InitHandshakeResult represents the response for initiating a multi-protocol handshake.
type InitHandshakeResult struct {
	Addrs            []string          `json:"addrs"`              // Shared node remote addresses (IPv4, IPv6, or domain names).
	AddPeerResponses []AddPeerResponse `json:"add_peer_responses"` // One response per requested protocol; Err set on per-item failure.
}

// InitHandshake signs and sends a multi-protocol handshake for the given session.
// Each entry in reqs carries a service Type and the already-JSON-marshaled
// protocol-specific peer request payload in Data. The message is signed once
// over Msg() (which binds the ID and every element's Type and Data).
func (c *Client) InitHandshake(ctx context.Context, id uint64, reqs []PeerRequest) (res *InitHandshakeResult, err error) {
	// Initialize the request body with the session ID and peer requests.
	req := &InitHandshakeRequestBody{
		ID:           id,
		PeerRequests: reqs,
	}

	// Sign the handshake message using the client's private key.
	signature, pubKey, err := c.Sign(c.fromName, req.Msg())
	if err != nil {
		return nil, fmt.Errorf("signing session %d data: %w", id, err)
	}

	// Set the public key and Base64-encoded signature in the request.
	req.PubKey = utils.EncodePubKey(pubKey)
	req.Signature = base64.StdEncoding.EncodeToString(signature)

	// Retrieve the API endpoint URL for the handshake.
	path, err := c.getURL(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("getting node API URL: %w", err)
	}

	// Send the HTTP POST request to initiate the handshake.
	res = &InitHandshakeResult{}
	if err := c.do(ctx, http.MethodPost, path, req, &res); err != nil {
		return nil, fmt.Errorf("performing init handshake request for session %d: %w", id, err)
	}

	// Return the response containing the handshake result.
	return res, nil
}
