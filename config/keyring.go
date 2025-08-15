package config

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/pflag"
)

// KeyringConfig represents the configuration for a keyring.
type KeyringConfig struct {
	Backend string    `mapstructure:"backend"` // Backend specifies the keyring backend to use.
	HomeDir string    `mapstructure:"-"`       // HomeDir is an optional home directory for the keyring.
	Input   io.Reader `mapstructure:"-"`       // Input is an optional reader for input data (not persisted).
	Name    string    `mapstructure:"name"`    // Name is the name of the keyring.
}

// GetBackend returns the keyring backend.
func (c *KeyringConfig) GetBackend() string {
	return c.Backend
}

// GetHomeDir returns the home directory.
func (c *KeyringConfig) GetHomeDir() string {
	return c.HomeDir
}

// GetInput returns the input reader.
func (c *KeyringConfig) GetInput() io.Reader {
	return c.Input
}

// GetName returns the keyring name.
func (c *KeyringConfig) GetName() string {
	return c.Name
}

// Validate validates the Keyring configuration.
func (c *KeyringConfig) Validate() error {
	// Check if the backend is one of the allowed values.
	validBackends := map[string]bool{
		"file":    true,
		"kwallet": true,
		"memory":  true,
		"os":      true,
		"pass":    true,
		"test":    true,
	}
	if !validBackends[c.Backend] {
		return fmt.Errorf("unsupported backend %q (allowed: file, kwallet, memory, os, pass, test)", c.Backend)
	}

	// Ensure the keyring name is not empty.
	if c.Name == "" {
		return errors.New("name cannot be empty")
	}

	return nil
}

// SetForFlags adds keyring configuration flags to the specified FlagSet.
func (c *KeyringConfig) SetForFlags(f *pflag.FlagSet) {
	f.StringVar(&c.Backend, "keyring.backend", c.Backend, "backend to use for the keyring (file, kwallet, memory, os, pass, test)")
	f.StringVar(&c.Name, "keyring.name", c.Name, "name identifier for the keyring")
}

// DefaultKeyringConfig returns the default Keyring configuration.
func DefaultKeyringConfig() *KeyringConfig {
	return &KeyringConfig{
		Backend: "test",
		HomeDir: "",
		Input:   nil,
		Name:    "sentinel",
	}
}
