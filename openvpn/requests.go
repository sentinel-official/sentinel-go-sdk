package openvpn

import (
	"encoding/json"
	"fmt"

	"github.com/v2fly/v2ray-core/v5/common/uuid"
)

// PeerRequest represents a V2Ray peer request.
type PeerRequest struct {
	UUID uuid.UUID `json:"uuid"`
}

// ID returns a string identifier for the peer (UUID).
func (r *PeerRequest) ID() string {
	return r.UUID.String()
}

// Validate checks if the PeerRequest is valid.
func (r *PeerRequest) Validate() error {
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
