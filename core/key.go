package core

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	cosmossdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/go-bip39"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// NewMnemonic generates a new mnemonic phrase using bip39 with 256 bits of entropy.
// Returns the mnemonic or an error if the operation fails.
func (c *Client) NewMnemonic() (string, error) {
	// Generate new entropy for the mnemonic.
	entropy, err := bip39.NewEntropy(1 << 8)
	if err != nil {
		return "", fmt.Errorf("generating entropy: %w", err)
	}

	// Create a new mnemonic phrase from the entropy.
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("generating mnemonic: %w", err)
	}

	return mnemonic, nil
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
			return "", nil, fmt.Errorf("generating mnemonic for key %q: %w", name, err)
		}
	}

	// Set the default HD path if none is provided.
	if hdPath == "" {
		hdPath = hd.CreateHDPath(cosmossdk.CoinType, 0, 0).String()
	}

	// Create a new key in the keyring.
	key, err := c.keyring.NewAccount(name, mnemonic, bip39Pass, hdPath, hd.Secp256k1)
	if err != nil {
		return "", nil, fmt.Errorf("creating key %q in keyring: %w", name, err)
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
		return fmt.Errorf("deleting key %q from keyring: %w", name, err)
	}

	return nil
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
		if utils.ErrorIs(err, errors.ErrKeyNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("retrieving key %q from keyring: %w", name, err)
	}

	return key, nil
}

// KeyForAddr retrieves the key record associated with the given account address from the keyring.
// Returns the key record or an error if the key cannot be found.
func (c *Client) KeyForAddr(addr cosmossdk.AccAddress) (*keyring.Record, error) {
	key, err := c.keyring.KeyByAddress(addr)
	if err != nil {
		if utils.ErrorIs(err, errors.ErrKeyNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("retrieving key for addr %q from keyring: %w", addr, err)
	}

	return key, nil
}

// Keys retrieves a list of all keys from the keyring.
// Returns the list of key records or an error if the operation fails.
func (c *Client) Keys() ([]*keyring.Record, error) {
	keys, err := c.keyring.List()
	if err != nil {
		return nil, fmt.Errorf("retrieving keys from keyring: %w", err)
	}

	return keys, nil
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
		return nil, nil, fmt.Errorf("signing data with key %q: %w", name, err)
	}

	return signature, pubKey, nil
}

// HasKey checks if a key exists in the keyring.
func (c *Client) HasKey(name string) (bool, error) {
	key, err := c.Key(name)
	if err != nil {
		return false, fmt.Errorf("getting key %q: %w", name, err)
	}
	if key == nil {
		return false, nil
	}

	return true, nil
}

// KeyAddr retrieves the key associated with the client's transaction signing identity
// and returns its corresponding address.
func (c *Client) KeyAddr(name string) (cosmossdk.AccAddress, error) {
	key, err := c.Key(name)
	if err != nil {
		return nil, fmt.Errorf("getting key %q: %w", name, err)
	}
	if key == nil {
		return nil, nil
	}

	// Obtain and return the address from the key.
	addr, err := key.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("getting addr from key %q: %w", name, err)
	}

	return addr, nil
}
