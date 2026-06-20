package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// PeerRequest wraps one protocol-specific peer request with a self-describing service type.
type PeerRequest struct {
	Data []byte `json:"data"` // JSON-marshaled protocol-specific peer request, must be non-empty.
	Type string `json:"type"` // Service type (e.g. "wireguard", "amneziawg", "hysteria2").
}

// AddPeerResponse wraps one protocol-specific add-peer response with a self-describing service type.
type AddPeerResponse struct {
	Data []byte `json:"data"` // JSON-marshaled protocol-specific response on success; empty on failure.
	Err  string `json:"err"`  // Non-empty error string on failure; empty on success.
	Type string `json:"type"` // Service type matching the corresponding PeerRequest.
}

// InitHandshakeRequestBody represents the request payload for adding a session.
type InitHandshakeRequestBody struct {
	Data      []byte `binding:"required,gt=0"        json:"data"`      // JSON encoded session data, must be present and non-empty.
	ID        uint64 `binding:"required,gt=0"        json:"id"`        // Unique identifier for the session, must be greater than zero.
	PubKey    string `binding:"required,gt=0"        json:"pub_key"`   // Public key associated with the session, required and non-empty.
	Signature string `binding:"required,base64,gt=0" json:"signature"` // Digital signature in Base64 format, required and non-empty.
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

// Msg constructs the message for signing by combining the session ID and data.
func (r *InitHandshakeRequestBody) Msg() (buf []byte) {
	buf = append(buf, types.Uint64ToBigEndian(r.ID)...)
	buf = append(buf, r.Data...)

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

// InitHandshakeResult represents the response for adding a session.
type InitHandshakeResult struct {
	Addrs []string `json:"addrs"` // List of addresses (IPv4, IPv6, or domain names).
	Data  []byte   `json:"data"`  // JSON encoded raw data containing additional response data.
}

// InitHandshake adds a session to a node by signing the session data and sending it to the node's API.
func (c *Client) InitHandshake(ctx context.Context, id uint64, data any) (res *InitHandshakeResult, err error) {
	// Initialize the request body with session ID.
	req := &InitHandshakeRequestBody{
		ID: id,
	}

	// Encode session data into JSON format.
	if req.Data, err = json.Marshal(data); err != nil {
		return nil, fmt.Errorf("encoding session %d data: %w", id, err)
	}

	// Sign the session message using the client's private key.
	signature, pubKey, err := c.Sign(c.fromName, req.Msg())
	if err != nil {
		return nil, fmt.Errorf("signing session %d data: %w", id, err)
	}

	// Set the public key and Base64-encoded signature in the request.
	req.PubKey = utils.EncodePubKey(pubKey)
	req.Signature = base64.StdEncoding.EncodeToString(signature)

	// Retrieve the API endpoint URL for adding a session.
	path, err := c.getURL(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("getting node API URL: %w", err)
	}

	// Send the HTTP POST request to add the session.
	res = &InitHandshakeResult{}
	if err := c.do(ctx, http.MethodPost, path, req, &res); err != nil {
		return nil, fmt.Errorf("performing init handshake request for session %d: %w", id, err)
	}

	// Return the response containing session details.
	return res, nil
}
