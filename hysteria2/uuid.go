package hysteria2

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewUUID generates and returns a new random UUID (v4) as a string.
func NewUUID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(fmt.Errorf("generating UUID: %w", err))
	}

	// Set version 4 and variant bits per RFC 4122.
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(buf[0:4]),
		hex.EncodeToString(buf[4:6]),
		hex.EncodeToString(buf[6:8]),
		hex.EncodeToString(buf[8:10]),
		hex.EncodeToString(buf[10:16]),
	)
}
