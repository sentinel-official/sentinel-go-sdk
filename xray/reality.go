package xray

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/sentinel-official/sentinel-go-sdk/v2/utils"
)

// Default Reality configuration values.
const (
	DefaultRealityDest        = "www.apple.com:443" // DefaultRealityDest is the default Reality dial-out target.
	DefaultRealityFingerprint = "chrome"            // DefaultRealityFingerprint is the default uTLS fingerprint.
)

// shortIDLength is the byte length of a Reality shortId.
const shortIDLength = 8

// Reality represents the Reality transport security configuration options.
type Reality struct {
	Dest        string   `mapstructure:"dest"`         // Dest defines the Reality dial-out target.
	ServerNames []string `mapstructure:"server_names"` // ServerNames defines the accepted SNI values.
	ShortIds    []string `mapstructure:"short_ids"`    // ShortIds defines the accepted shortId values.
	Fingerprint string   `mapstructure:"fingerprint"`  // Fingerprint defines the uTLS fingerprint.
	PrivateKey  string   `mapstructure:"private_key"`  // PrivateKey is the generated X25519 private key.
	PublicKey   string   `mapstructure:"public_key"`   // PublicKey is the derived X25519 public key.
}

// Validate validates the Reality fields.
func (c *Reality) Validate() error {
	// Ensure Dest is not empty.
	if c.Dest == "" {
		return errors.New("dest is empty")
	}

	// Ensure ServerNames is not empty.
	if len(c.ServerNames) == 0 {
		return errors.New("server_names are empty")
	}

	// Ensure ShortIds is not empty.
	if len(c.ShortIds) == 0 {
		return errors.New("short_ids are empty")
	}

	// Validate each shortId is a valid hex value.
	for _, shortID := range c.ShortIds {
		b, err := hex.DecodeString(shortID)
		if err != nil {
			return fmt.Errorf("parsing short_id %q: %w", shortID, err)
		}

		if len(b) > shortIDLength {
			return fmt.Errorf("short_id %q exceeds %d bytes", shortID, shortIDLength)
		}
	}

	// Ensure Fingerprint is not empty.
	if c.Fingerprint == "" {
		return errors.New("fingerprint is empty")
	}

	// Ensure PrivateKey is not empty.
	if c.PrivateKey == "" {
		return errors.New("private_key is empty")
	}

	// Ensure PublicKey is not empty.
	if c.PublicKey == "" {
		return errors.New("public_key is empty")
	}

	// Reject values that could break out of a JSON string literal in the config.
	for _, v := range append([]string{c.Dest}, c.ServerNames...) {
		if utils.HasJSONUnsafeChars(v) {
			return fmt.Errorf("field contains unsafe characters: %q", v)
		}
	}

	return nil
}

// generateKeys generates a new X25519 keypair and stores them in the Reality config.
func (c *Reality) generateKeys() error {
	// Generate a new X25519 private key.
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generating private key: %w", err)
	}

	c.PrivateKey = base64.RawURLEncoding.EncodeToString(key.Bytes())
	c.PublicKey = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())

	return nil
}

// newShortID generates a new random Reality shortId.
func newShortID() (string, error) {
	buf := make([]byte, shortIDLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating short_id: %w", err)
	}

	return hex.EncodeToString(buf), nil
}
