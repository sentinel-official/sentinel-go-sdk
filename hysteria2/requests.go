package hysteria2

import (
	"encoding/json"
	"errors"
	"fmt"
)

// PeerRequest represents a Hysteria2 peer request.
type PeerRequest struct {
	UUID string `json:"uuid"`
}

// ID returns a string identifier for the peer (UUID).
func (r *PeerRequest) ID() string {
	return r.UUID
}

// Validate checks if the PeerRequest is valid.
func (r *PeerRequest) Validate() error {
	if r.UUID == "" {
		return errors.New("uuid is empty")
	}

	return nil
}

// parsePeerRequest attempts to parse input into a *PeerRequest.
// - If input is []byte, it unmarshals it as JSON.
// - If input is already a *PeerRequest, it returns it.
// - Otherwise, it returns an error.
func parsePeerRequest(input any) (*PeerRequest, error) {
	switch v := input.(type) {
	case []byte:
		var req PeerRequest
		if err := json.Unmarshal(v, &req); err != nil {
			return nil, fmt.Errorf("unmarshaling input: %w", err)
		}

		return &req, nil
	case *PeerRequest:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported input type %T", input)
	}
}
