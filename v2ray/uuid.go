package v2ray

import (
	"github.com/google/uuid"
)

// NewUUID generates and returns a new UUID.
func NewUUID() uuid.UUID {
	return uuid.New()
}

// NewStringUUID generates a new UUID and returns it as a string.
func NewStringUUID() string {
	return uuid.NewString()
}
