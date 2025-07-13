package openvpn

import (
	"encoding/json"
	"fmt"

	"github.com/v2fly/v2ray-core/v5/common/uuid"
)

// ServiceRequest represents a OpenVPN service request.
type ServiceRequest struct {
	UUID uuid.UUID `json:"uuid"`
}

// Validate checks if the ServiceRequest is valid.
func (r *ServiceRequest) Validate() error {
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
