package wireguard

import (
	"errors"
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
