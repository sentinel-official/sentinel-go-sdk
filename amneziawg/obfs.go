package amneziawg

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	obfsHMin  = 5     // minimum value for H1–H4 magic headers (exclusive of 4)
	obfsHMax  = 65535 // maximum value for H1–H4 magic headers
	obfsSMax  = 65535 // maximum value for S1–S4
	obfsIMax  = 65535 // maximum value for I1–I5
	obfsJcMax = 128   // maximum value for Jc (junk packet count)
	obfsJMin  = 1     // minimum value for Jmin
	obfsJMax  = 1280  // maximum value for Jmax
)

// Obfs holds the AmneziaWG interface-level obfuscation parameters.
//
// Two parameter classes exist:
//   - Handshake-affecting (must match on both ends): S1–S4, H1–H4, I1–I5.
//   - Local-only junk (may differ per peer): Jc, Jmin, Jmax.
//
// AdvancedSecurity is a per-peer flag enabling additional handshake protection.
type Obfs struct {
	// Junk packet parameters — local only, need not match the remote peer.
	Jc   uint8  `mapstructure:"jc"`   // Jc is the number of junk packets to send (0–128).
	Jmin uint16 `mapstructure:"jmin"` // Jmin is the minimum junk packet size in bytes.
	Jmax uint16 `mapstructure:"jmax"` // Jmax is the maximum junk packet size in bytes.

	// Handshake parameters — must match on both endpoints.
	S1 uint16 `mapstructure:"s1"` // S1 is the first handshake obfuscation size.
	S2 uint16 `mapstructure:"s2"` // S2 is the second handshake obfuscation size.
	S3 uint16 `mapstructure:"s3"` // S3 is the third handshake obfuscation size.
	S4 uint16 `mapstructure:"s4"` // S4 is the fourth handshake obfuscation size.

	H1 uint32 `mapstructure:"h1"` // H1 is the first magic header (distinct, > 4).
	H2 uint32 `mapstructure:"h2"` // H2 is the second magic header (distinct, > 4).
	H3 uint32 `mapstructure:"h3"` // H3 is the third magic header (distinct, > 4).
	H4 uint32 `mapstructure:"h4"` // H4 is the fourth magic header (distinct, > 4).

	I1 uint32 `mapstructure:"i1"` // I1 is the first imitation/signature packet value.
	I2 uint32 `mapstructure:"i2"` // I2 is the second imitation/signature packet value.
	I3 uint32 `mapstructure:"i3"` // I3 is the third imitation/signature packet value.
	I4 uint32 `mapstructure:"i4"` // I4 is the fourth imitation/signature packet value.
	I5 uint32 `mapstructure:"i5"` // I5 is the fifth imitation/signature packet value.

	// Per-peer flag.
	AdvancedSecurity bool `mapstructure:"advanced_security"` // AdvancedSecurity enables additional handshake protection.
}

// randUint32 returns a cryptographically random uint32.
func randUint32() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("reading random bytes: %w", err)
	}

	return binary.LittleEndian.Uint32(b[:]), nil
}

// randUint16 returns a cryptographically random uint16.
func randUint16() (uint16, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("reading random bytes: %w", err)
	}

	return binary.LittleEndian.Uint16(b[:]), nil
}

// Generate randomizes a new Obfs profile.
//
// H1–H4 are generated as distinct uint32 values each greater than 4.
// S1–S4 and I1–I5 are randomized uint16/uint32 values.
// Jc/Jmin/Jmax are set to sensible defaults (7, 50, 1000).
func (o *Obfs) Generate() error {
	// Generate H1–H4 as distinct values each > 4.
	seen := make(map[uint32]bool, 4)

	hFields := []*uint32{&o.H1, &o.H2, &o.H3, &o.H4}
	for _, h := range hFields {
		for {
			v, err := randUint32()
			if err != nil {
				return fmt.Errorf("generating magic header: %w", err)
			}

			// Ensure value is > 4 and not already used.
			v = v%(obfsHMax-obfsHMin) + obfsHMin + 1
			if !seen[v] {
				*h = v
				seen[v] = true

				break
			}
		}
	}

	// Generate S1–S4 (any uint16 value is valid).
	sFields := []*uint16{&o.S1, &o.S2, &o.S3, &o.S4}
	for _, s := range sFields {
		v, err := randUint16()
		if err != nil {
			return fmt.Errorf("generating S parameter: %w", err)
		}

		*s = v
	}

	// Generate I1–I5 (values in range 0–65535).
	iFields := []*uint32{&o.I1, &o.I2, &o.I3, &o.I4, &o.I5}
	for _, i := range iFields {
		v, err := randUint32()
		if err != nil {
			return fmt.Errorf("generating I parameter: %w", err)
		}

		*i = v % (obfsIMax + 1)
	}

	// Set sensible junk defaults.
	o.Jc = 7
	o.Jmin = 50
	o.Jmax = 1000

	o.AdvancedSecurity = true

	return nil
}

// Validate checks that all Obfs fields satisfy the constraints from amneziawg-tools/src/config.c.
//
//   - H1–H4 must be distinct and each > 4.
//   - Jmin < Jmax.
func (o *Obfs) Validate() error {
	// Validate Jc.
	if o.Jc > obfsJcMax {
		return fmt.Errorf("jc %d exceeds maximum %d", o.Jc, obfsJcMax)
	}

	// Validate Jmin < Jmax.
	if o.Jmin >= o.Jmax {
		return errors.New("jmin must be less than jmax")
	}

	// Validate Jmin and Jmax bounds.
	if o.Jmin < obfsJMin {
		return fmt.Errorf("jmin %d is below minimum %d", o.Jmin, obfsJMin)
	}

	if o.Jmax > obfsJMax {
		return fmt.Errorf("jmax %d exceeds maximum %d", o.Jmax, obfsJMax)
	}

	// Validate H1–H4: each must be > 4 and all must be distinct.
	hs := [4]uint32{o.H1, o.H2, o.H3, o.H4}
	for i, h := range hs {
		if h <= 4 {
			return fmt.Errorf("h%d must be greater than 4", i+1)
		}

		if h > obfsHMax {
			return fmt.Errorf("h%d %d exceeds maximum %d", i+1, h, obfsHMax)
		}
	}

	seen := make(map[uint32]bool, 4)
	for i, h := range hs {
		if seen[h] {
			return fmt.Errorf("h%d %d is not distinct", i+1, h)
		}

		seen[h] = true
	}

	return nil
}
