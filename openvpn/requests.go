package openvpn

import (
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
