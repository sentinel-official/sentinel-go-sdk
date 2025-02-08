package v2ray

import (
	"github.com/v2fly/v2ray-core/v5/common/uuid"
)

// NewUUID generates and returns a new UUID.
func NewUUID() uuid.UUID {
	return uuid.New()
}

// NewStringUUID generates a new UUID and returns it as a string.
func NewStringUUID() string {
	i := NewUUID()
	return i.String()
}
