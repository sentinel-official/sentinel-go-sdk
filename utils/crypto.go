package utils

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/types"
)

// EncodePubKey encodes a public key to a base64-formatted string with its type.
func EncodePubKey(key types.PubKey) string {
	if key == nil {
		return ""
	}

	return fmt.Sprintf("%s:%s", key.Type(), base64.StdEncoding.EncodeToString(key.Bytes()))
}

// DecodePubKey decodes a base64-formatted string into a public key.
func DecodePubKey(s string) (types.PubKey, error) {
	// Remove extra spaces
	s = strings.TrimSpace(s)

	// Split into type and key parts
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("got %d fields, expected 2 (type:key)", len(parts))
	}

	// Decode base64 key
	key, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding key part: %w", err)
	}

	switch parts[0] {
	case "ed25519":
		return decodeEd25519Key(key)
	case "secp256k1":
		return decodeSecp256k1Key(key)
	default:
		return nil, fmt.Errorf("unsupported public key type %q", parts[0])
	}
}

// decodeEd25519Key validates and decodes an Ed25519 public key.
func decodeEd25519Key(keyBytes []byte) (types.PubKey, error) {
	if len(keyBytes) != ed25519.PubKeySize {
		return nil, fmt.Errorf("invalid ed25519 public key size %d, expected %d", len(keyBytes), ed25519.PubKeySize)
	}

	return &ed25519.PubKey{Key: keyBytes}, nil
}

// decodeSecp256k1Key validates and decodes a Secp256k1 public key.
func decodeSecp256k1Key(keyBytes []byte) (types.PubKey, error) {
	if len(keyBytes) != secp256k1.PubKeySize {
		return nil, fmt.Errorf("invalid secp256k1 public key size %d, expected %d", len(keyBytes), secp256k1.PubKeySize)
	}

	return &secp256k1.PubKey{Key: keyBytes}, nil
}
