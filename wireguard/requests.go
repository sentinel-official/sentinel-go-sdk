package wireguard

import (
	"encoding/json"
	"errors"
	"fmt"
)

// PeerRequest represents a WireGuard peer request.
type PeerRequest struct {
	PublicKey *Key `json:"public_key"`
}

// ID returns a string identifier for the peer, typically the public key in base64 or hex format.
func (r *PeerRequest) ID() string {
	if r.PublicKey == nil {
		return ""
	}

	return r.PublicKey.String()
}

// Validate checks if the PeerRequest is valid.
func (r *PeerRequest) Validate() error {
	if r.PublicKey == nil {
		return errors.New("public_key cannot be nil")
	}
	if r.PublicKey.IsZero() {
		return errors.New("public_key cannot be zero")
	}

	return nil
}

// parsePeerRequest attempts to parse input into a *PeerRequest.
// - If input is []byte, it unmarshals it as JSON.
// - If input is already a *PeerRequest, it returns it.
// - Otherwise, it returns an error.
func parsePeerRequest(input interface{}) (*PeerRequest, error) {
	switch v := input.(type) {
	case []byte:
		var req PeerRequest
		if err := json.Unmarshal(v, &req); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input: %w", err)
		}

		return &req, nil
	case *PeerRequest:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported input type %T", input)
	}
}
