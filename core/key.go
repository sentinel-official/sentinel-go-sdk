package core

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/go-bip39"
)

// KeyForAddr retrieves the key record associated with the given account address from the keyring.
// Returns the key record or an error if the key cannot be found.
func (c *Client) KeyForAddr(addr cosmossdk.AccAddress) (*keyring.Record, error) {
	key, err := c.keyring.KeyByAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get key for addr from keyring: %w", err)
	}

	return key, nil
}

// CreateKey generates and stores a new key in the keyring with the provided name, mnemonic, and options.
// If no mnemonic is provided, it generates a new one.
// Returns the mnemonic, the created key record, and any error encountered.
func (c *Client) CreateKey(name, mnemonic, bip39Pass, hdPath string) (s string, k *keyring.Record, err error) {
	// Use the default transaction key name if none is provided.
	if name == "" {
		name = c.txFromName
	}

	// Generate a new mnemonic if none is provided.
	if mnemonic == "" {
		mnemonic, err = c.NewMnemonic()
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate new mnemonic: %w", err)
		}
	}

	// Set the default HD path if none is provided.
	if hdPath == "" {
		hdPath = hd.CreateHDPath(cosmossdk.CoinType, 0, 0).String()
	}

	// Create a new key in the keyring.
	key, err := c.keyring.NewAccount(name, mnemonic, bip39Pass, hdPath, hd.Secp256k1)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create new account: %w", err)
	}

	return mnemonic, key, nil
}

// DeleteKey removes a key from the keyring based on the provided name.
// Returns an error if the key cannot be deleted.
func (c *Client) DeleteKey(name string) error {
	// Use the default transaction key name if none is provided.
	if name == "" {
		name = c.txFromName
	}

	if err := c.keyring.Delete(name); err != nil {
		return fmt.Errorf("failed to delete key from keyring: %w", err)
	}

	return nil
}

// HasKey checks if a key exists in the keyring.
func (c *Client) HasKey(name string) (bool, error) {
	key, err := c.Key(name)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve key: %w", err)
	}

	return key != nil, nil
}

// Key retrieves key information from the keyring based on the provided name.
// Returns the key record or an error if the key cannot be found.
func (c *Client) Key(name string) (*keyring.Record, error) {
	// Use the default transaction key name if none is provided.
	if name == "" {
		name = c.txFromName
	}

	key, err := c.keyring.Key(name)
	if err != nil {
		if errors.IsOf(err, errors.ErrKeyNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to retrieve key from keyring: %w", err)
	}

	return key, nil
}

// KeyAddr retrieves the key associated with the client's transaction signing identity
// and returns its corresponding address.
func (c *Client) KeyAddr(name string) (cosmossdk.AccAddress, error) {
	key, err := c.Key(name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key: %w", err)
	}
	if key == nil {
		return nil, nil
	}

	// Obtain and return the address from the key.
	addr, err := key.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get addr from key: %w", err)
	}

	return addr, nil
}

// Keys retrieves a list of all keys from the keyring.
// Returns the list of key records or an error if the operation fails.
func (c *Client) Keys() ([]*keyring.Record, error) {
	keys, err := c.keyring.List()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve keys from keyring: %w", err)
	}

	return keys, nil
}

// NewMnemonic generates a new mnemonic phrase using bip39 with 256 bits of entropy.
// Returns the mnemonic or an error if the operation fails.
func (c *Client) NewMnemonic() (string, error) {
	// Generate new entropy for the mnemonic.
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	// Create a new mnemonic phrase from the entropy.
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	return mnemonic, nil
}

// Sign signs the provided data using the key from the keyring identified by the given name.
// Returns the signed bytes, the public key, and any error encountered.
func (c *Client) Sign(name string, buf []byte) ([]byte, types.PubKey, error) {
	// Use the default transaction key name if none is provided.
	if name == "" {
		name = c.txFromName
	}

	signature, pubKey, err := c.keyring.Sign(name, buf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign data: %w", err)
	}

	return signature, pubKey, nil
}
