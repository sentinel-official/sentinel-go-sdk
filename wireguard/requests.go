package wireguard

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ServiceRequest represents a WireGuard service request.
type ServiceRequest struct {
	PublicKey *Key `json:"public_key"`
}

// Validate checks if the ServiceRequest is valid.
func (r *ServiceRequest) Validate() error {
	if r.PublicKey == nil {
		return errors.New("public_key cannot be nil")
	}
	if r.PublicKey.IsZero() {
		return errors.New("public_key cannot be zero")
	}

	return nil
}

// parseServiceRequest attempts to parse input into a *ServiceRequest.
// - If input is []byte, it unmarshals it as JSON.
// - If input is already a *ServiceRequest, it returns it.
// - Otherwise, it returns an error.
func parseServiceRequest(input interface{}) (*ServiceRequest, error) {
	switch v := input.(type) {
	case []byte:
		var req ServiceRequest
		if err := json.Unmarshal(v, &req); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input: %w", err)
		}

		return &req, nil
	case *ServiceRequest:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported input type %T", input)
	}
}
