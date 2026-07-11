package uuid

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// UUID wraps google/uuid.UUID with JSON encoding compatible with both the
// legacy byte-array form and the canonical string form.
type UUID uuid.UUID //nolint:recvcheck

// New generates a random UUID.
func New() UUID {
	return UUID(uuid.New())
}

// Parse decodes a canonical UUID string.
func Parse(s string) (UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, fmt.Errorf("parsing uuid %q: %w", s, err)
	}

	return UUID(u), nil
}

// Raw returns the underlying google/uuid.UUID.
func (u UUID) Raw() uuid.UUID {
	return uuid.UUID(u)
}

// String returns the canonical string representation.
func (u UUID) String() string {
	return uuid.UUID(u).String()
}

// MarshalJSON encodes the UUID as a byte array for legacy compatibility.
func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal([16]byte(u)) //nolint:wrapcheck
}

// UnmarshalJSON decodes the UUID from either a canonical string or a byte array.
func (u *UUID) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)

	if len(trimmed) > 0 && trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return fmt.Errorf("unmarshaling uuid string: %w", err)
		}

		parsed, err := uuid.Parse(s)
		if err != nil {
			return fmt.Errorf("parsing uuid %q: %w", s, err)
		}

		*u = UUID(parsed)

		return nil
	}

	var arr [16]byte
	if err := json.Unmarshal(trimmed, &arr); err != nil {
		return fmt.Errorf("unmarshaling uuid array: %w", err)
	}

	*u = UUID(arr)

	return nil
}
