package amneziawg

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	obfsJcMax    = 10   // maximum value for Jc (junk packet count)
	obfsJSizeMin = 64   // minimum value for Jmin/Jmax (bytes)
	obfsJSizeMax = 1024 // maximum value for Jmin/Jmax (bytes)
	obfsS123Max  = 64   // maximum value for S1, S2, S3 (junk-prefix byte sizes)
	obfsS4Max    = 32   // maximum value for S4 (junk-prefix byte size)
	obfsSDelta   = 56   // size gap between the padded init (148) and response (92) handshake packets
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
	Jc   uint8  `mapstructure:"jc"`   // Jc is the number of junk packets to send (0–10).
	Jmin uint16 `mapstructure:"jmin"` // Jmin is the minimum junk packet size in bytes (64–1024).
	Jmax uint16 `mapstructure:"jmax"` // Jmax is the maximum junk packet size in bytes (64–1024).

	// Handshake parameters — must match on both endpoints.
	S1 uint16 `mapstructure:"s1"` // S1 is the first handshake obfuscation size (0–64).
	S2 uint16 `mapstructure:"s2"` // S2 is the second handshake obfuscation size (0–64).
	S3 uint16 `mapstructure:"s3"` // S3 is the third handshake obfuscation size (0–64).
	S4 uint16 `mapstructure:"s4"` // S4 is the fourth handshake obfuscation size (0–32).

	H1 uint32 `mapstructure:"h1"` // H1 is the first magic header (distinct, > 4).
	H2 uint32 `mapstructure:"h2"` // H2 is the second magic header (distinct, > 4).
	H3 uint32 `mapstructure:"h3"` // H3 is the third magic header (distinct, > 4).
	H4 uint32 `mapstructure:"h4"` // H4 is the fourth magic header (distinct, > 4).

	// I1–I5 are optional Custom Protocol Signature strings. When non-empty they
	// must match on both endpoints. Valid values are hex-blob strings with tags
	// such as <b>, <t>, <r>, <rc>, <rd> as defined by amneziawg-tools/src/config.c.
	I1 string `mapstructure:"i1"` // I1 is the first custom protocol signature string.
	I2 string `mapstructure:"i2"` // I2 is the second custom protocol signature string.
	I3 string `mapstructure:"i3"` // I3 is the third custom protocol signature string.
	I4 string `mapstructure:"i4"` // I4 is the fourth custom protocol signature string.
	I5 string `mapstructure:"i5"` // I5 is the fifth custom protocol signature string.

	// Per-peer flag.
	AdvancedSecurity bool `mapstructure:"advanced_security"` // AdvancedSecurity enables additional handshake protection (kernel-module only).
}

// randUint32 returns a cryptographically random uint32.
func randUint32() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("reading random bytes: %w", err)
	}

	return binary.LittleEndian.Uint32(b[:]), nil
}

// randUint16n returns a cryptographically random uint16 in [0, n].
func randUint16n(n uint16) (uint16, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("reading random bytes: %w", err)
	}

	return binary.LittleEndian.Uint16(b[:]) % (n + 1), nil
}

// Generate randomizes a new Obfs profile.
//
// H1–H4 are generated as distinct uint32 values each greater than 4.
// S1–S3 are randomized in [0, 64]; S4 in [0, 32].
// Jc/Jmin/Jmax are randomized within their valid ranges.
// I1–I5 are left empty — valid CPS signatures cannot be meaningfully randomized.
// AdvancedSecurity defaults to false.
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
			if v <= 4 {
				continue
			}

			if !seen[v] {
				*h = v
				seen[v] = true

				break
			}
		}
	}

	// Generate S1–S3 in [0, 64].
	for _, s := range []*uint16{&o.S1, &o.S2, &o.S3} {
		v, err := randUint16n(obfsS123Max)
		if err != nil {
			return fmt.Errorf("generating S parameter: %w", err)
		}

		*s = v
	}

	// amneziawg rejects S1 + 56 == S2 (padded init and response packets would
	// share a size); regenerate S2 until they differ.
	for o.S1+obfsSDelta == o.S2 {
		v, err := randUint16n(obfsS123Max)
		if err != nil {
			return fmt.Errorf("generating S2 parameter: %w", err)
		}

		o.S2 = v
	}

	// Generate S4 in [0, 32].
	v, err := randUint16n(obfsS4Max)
	if err != nil {
		return fmt.Errorf("generating S4 parameter: %w", err)
	}

	o.S4 = v

	// Randomize junk-packet parameters so each install has a distinct junk
	// fingerprint. Jmin stays below Jmax by construction.
	jc, err := randUint16n(obfsJcMax - 3)
	if err != nil {
		return fmt.Errorf("generating jc parameter: %w", err)
	}

	o.Jc = uint8(jc) + 3

	jmin, err := randUint16n(192)
	if err != nil {
		return fmt.Errorf("generating jmin parameter: %w", err)
	}

	o.Jmin = jmin + obfsJSizeMin

	jmax, err := randUint16n(512)
	if err != nil {
		return fmt.Errorf("generating jmax parameter: %w", err)
	}

	o.Jmax = jmax + 512

	// I1–I5 are left empty (CPS strings are not auto-generated).
	o.I1 = ""
	o.I2 = ""
	o.I3 = ""
	o.I4 = ""
	o.I5 = ""

	o.AdvancedSecurity = false

	return nil
}

// Validate checks that all Obfs fields satisfy the constraints from amneziawg-tools/src/config.c
// and the documented parameter ranges at docs.amnezia.org.
//
//   - Jc: 0–10.
//   - Jmin and Jmax: 64–1024; Jmin < Jmax.
//   - S1, S2, S3: 0–64; S4: 0–32.
//   - H1–H4: distinct and each > 4.
//   - I1–I5: optional strings, free of control characters and shell metacharacters.
func (o *Obfs) Validate() error {
	// Validate Jc.
	if o.Jc > obfsJcMax {
		return fmt.Errorf("jc %d exceeds maximum %d", o.Jc, obfsJcMax)
	}

	// Validate Jmin and Jmax bounds.
	if o.Jmin < obfsJSizeMin {
		return fmt.Errorf("jmin %d is below minimum %d", o.Jmin, obfsJSizeMin)
	}

	if o.Jmax > obfsJSizeMax {
		return fmt.Errorf("jmax %d exceeds maximum %d", o.Jmax, obfsJSizeMax)
	}

	// Validate Jmin < Jmax.
	if o.Jmin >= o.Jmax {
		return errors.New("jmin must be less than jmax")
	}

	// Validate S1–S3 (0–64).
	for i, s := range [3]uint16{o.S1, o.S2, o.S3} {
		if s > obfsS123Max {
			return fmt.Errorf("s%d %d exceeds maximum %d", i+1, s, obfsS123Max)
		}
	}

	// Validate S4 (0–32).
	if o.S4 > obfsS4Max {
		return fmt.Errorf("s4 %d exceeds maximum %d", o.S4, obfsS4Max)
	}

	// S1 + 56 must not equal S2 (amneziawg rejects equal-size init/response packets).
	if o.S1+obfsSDelta == o.S2 {
		return errors.New("s1 + 56 must not equal s2")
	}

	// Validate H1–H4: each must be > 4 and all must be distinct.
	hs := [4]uint32{o.H1, o.H2, o.H3, o.H4}
	for i, h := range hs {
		if h <= 4 {
			return fmt.Errorf("h%d must be greater than 4", i+1)
		}
	}

	seen := make(map[uint32]bool, 4)
	for i, h := range hs {
		if seen[h] {
			return fmt.Errorf("h%d %d is not distinct", i+1, h)
		}

		seen[h] = true
	}

	for i, s := range [5]string{o.I1, o.I2, o.I3, o.I4, o.I5} {
		for _, r := range s {
			if r < 0x20 || r == 0x7f || strings.ContainsRune("\"'`$;\\", r) {
				return fmt.Errorf("i%d contains an unsafe character", i+1)
			}
		}
	}

	return nil
}
