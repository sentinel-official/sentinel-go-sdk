package node

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// AddSessionRequestBody represents the request payload for adding a session.
type AddSessionRequestBody struct {
	Data      string `json:"data" binding:"required,base64,gt=0"`      // Encoded session data (Base64 format), must be present and non-empty.
	ID        uint64 `json:"id" binding:"required,gt=0"`               // Unique identifier for the session, must be greater than zero.
	PubKey    string `json:"pub_key" binding:"required,gt=0"`          // Public key associated with the session, required and non-empty.
	Signature string `json:"signature" binding:"required,base64,gt=0"` // Digital signature to verify the integrity, must be in Base64 format.
}

// AccAddr converts the public key into a Cosmos SDK AccAddress.
func (r *AddSessionRequestBody) AccAddr() (types.AccAddress, error) {
	// Decode the public key.
	pubKey, err := utils.DecodePubKey(r.PubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	return pubKey.Address().Bytes(), nil
}

// DecodeData decodes the Base64-encoded JSON string into the provided target structure.
func (r *AddSessionRequestBody) DecodeData(target interface{}) error {
	buf, err := base64.StdEncoding.DecodeString(r.Data)
	if err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}

	if err := json.Unmarshal(buf, target); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// EncodeData marshals the given data into JSON and encodes it in Base64.
func (r *AddSessionRequestBody) EncodeData(data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Encode JSON to Base64.
	r.Data = base64.StdEncoding.EncodeToString(buf)
	return nil
}

// Msg constructs the message for signing by combining the session ID and data.
func (r *AddSessionRequestBody) Msg() (buf []byte) {
	buf = append(buf, types.Uint64ToBigEndian(r.ID)...)
	buf = append(buf, r.Data...)
	return buf
}

// Verify checks whether the provided signature is valid for the given message and public key.
func (r *AddSessionRequestBody) Verify() error {
	// Decode the public key.
	pubKey, err := utils.DecodePubKey(r.PubKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	// Decode the signature from Base64.
	signature, err := base64.StdEncoding.DecodeString(r.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify the signature against the message and public key.
	if !pubKey.VerifySignature(r.Msg(), signature) {
		return errors.New("signature verification failed")
	}

	return nil
}

// AddSessionResult represents the response for adding a session.
type AddSessionResult struct {
	Addrs []string `json:"addrs"` // List of addresses (IPv4, IPv6, or domain names).
	Data  string   `json:"data"`  // Base64-encoded JSON string containing additional response data.
}

// DecodeData decodes the Base64-encoded JSON string into the provided target structure.
func (r *AddSessionResult) DecodeData(target interface{}) error {
	buf, err := base64.StdEncoding.DecodeString(r.Data)
	if err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}

	if err := json.Unmarshal(buf, target); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// EncodeData marshals the given data into JSON and encodes it in Base64.
func (r *AddSessionResult) EncodeData(data interface{}) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Encode JSON to Base64.
	r.Data = base64.StdEncoding.EncodeToString(buf)
	return nil
}

// AddSession adds a session to a node by signing the session data and sending it to the node's API.
func (c *Client) AddSession(ctx context.Context, id uint64, data interface{}) (*AddSessionResult, error) {
	// Initialize the request body with session ID.
	req := &AddSessionRequestBody{
		ID: id,
	}

	// Encode session data into Base64 format.
	if err := req.EncodeData(data); err != nil {
		return nil, fmt.Errorf("failed to encode data: %w", err)
	}

	// Sign the session message using the client's private key.
	signature, pubKey, err := c.Sign(c.fromName, req.Msg())
	if err != nil {
		return nil, fmt.Errorf("failed to sign session data: %w", err)
	}

	// Set the public key and Base64-encoded signature in the request.
	req.PubKey = utils.EncodePubKey(pubKey)
	req.Signature = base64.StdEncoding.EncodeToString(signature)

	// Retrieve the API endpoint URL for adding a session.
	path, err := c.getURL(ctx, "sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to get url: %w", err)
	}

	// Send the HTTP POST request to add the session.
	var res AddSessionResult
	if err := c.do(ctx, http.MethodPost, path, req, &res); err != nil {
		return nil, err
	}

	// Return the response containing session details.
	return &res, nil
}
