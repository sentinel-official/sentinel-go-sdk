package utils

import (
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
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
		return nil, errors.New("invalid public key format")
	}

	// Decode base64 key
	key, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to base64 key: %w", err)
	}

	switch parts[0] {
	case "ed25519":
		return decodeEd25519Key(key)
	case "secp256k1":
		return decodeSecp256k1Key(key)
	default:
		return nil, errors.New("unsupported public key type")
	}
}

// decodeEd25519Key validates and decodes an Ed25519 public key.
func decodeEd25519Key(keyBytes []byte) (types.PubKey, error) {
	if len(keyBytes) != ed25519.PubKeySize {
		return nil, errors.New("invalid ed25519 public key size")
	}

	return &ed25519.PubKey{Key: keyBytes}, nil
}

// decodeSecp256k1Key validates and decodes a Secp256k1 public key.
func decodeSecp256k1Key(keyBytes []byte) (types.PubKey, error) {
	if len(keyBytes) != secp256k1.PubKeySize {
		return nil, errors.New("invalid secp256k1 public key size")
	}

	return &secp256k1.PubKey{Key: keyBytes}, nil
}

// WritePEMFile writes a PEM-encoded block to the specified file path.
func WritePEMFile(path, blockType string, data []byte) error {
	// Create the file at the specified path
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	defer file.Close()

	// Create PEM block
	block := &pem.Block{
		Type:  blockType,
		Bytes: data,
	}

	// Encode the PEM block into the file
	if err := pem.Encode(file, block); err != nil {
		return fmt.Errorf("failed to encode pem block to file: %w", err)
	}

	return nil
}
