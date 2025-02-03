package core

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/sentinel-official/sentinel-go-sdk/config"
)

// SetupKeyring initializes and configures a keyring for cryptographic key management.
func (c *Client) SetupKeyring(cfg *config.KeyringConfig) error {
	// Create a keyring instance using the provided configuration.
	kr, err := keyring.New(cfg.GetName(), cfg.GetBackend(), cfg.GetHomeDir(), cfg.GetInput(), c.protoCodec)
	if err != nil {
		return fmt.Errorf("failed to create keyring: %w", err)
	}

	// Assign the created keyring to the client.
	c.WithKeyring(kr)
	return nil
}
