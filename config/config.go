package config

import (
	"fmt"

	"github.com/spf13/pflag"
)

// Config represents the overall configuration structure.
type Config struct {
	Keyring *KeyringConfig `mapstructure:"keyring"` // Keyring contains keyring configuration.
	Query   *QueryConfig   `mapstructure:"query"`   // Query contains query configuration.
	RPC     *RPCConfig     `mapstructure:"rpc"`     // RPC contains RPC configuration.
	Tx      *TxConfig      `mapstructure:"tx"`      // Tx contains transaction configuration.
}

// Validate validates the entire configuration.
func (c *Config) Validate() error {
	if err := c.Keyring.Validate(); err != nil {
		return fmt.Errorf("validating keyring config: %w", err)
	}

	if err := c.Query.Validate(); err != nil {
		return fmt.Errorf("validating query config: %w", err)
	}

	if err := c.RPC.Validate(); err != nil {
		return fmt.Errorf("validating rpc config: %w", err)
	}

	if err := c.Tx.Validate(); err != nil {
		return fmt.Errorf("validating tx config: %w", err)
	}

	return nil
}

// SetForFlags adds configuration flags to the specified FlagSet.
func (c *Config) SetForFlags(f *pflag.FlagSet) {
	c.Keyring.SetForFlags(f)
	c.Query.SetForFlags(f)
	c.RPC.SetForFlags(f)
	c.Tx.SetForFlags(f)
}

// DefaultConfig returns a configuration instance with default values.
func DefaultConfig() *Config {
	return &Config{
		Keyring: DefaultKeyringConfig(),
		Query:   DefaultQueryConfig(),
		RPC:     DefaultRPCConfig(),
		Tx:      DefaultTxConfig(),
	}
}
