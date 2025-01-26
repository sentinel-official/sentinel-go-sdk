package config

import (
	"fmt"

	"github.com/spf13/pflag"
)

// Config represents the overall configuration structure.
type Config struct {
	Keyring *KeyringConfig `mapstructure:"keyring"` // Keyring contains keyring configuration.
	Log     *LogConfig     `mapstructure:"log"`     // Log contains logging configuration.
	Query   *QueryConfig   `mapstructure:"query"`   // Query contains query configuration.
	RPC     *RPCConfig     `mapstructure:"rpc"`     // RPC contains RPC configuration.
	Tx      *TxConfig      `mapstructure:"tx"`      // Tx contains transaction configuration.
}

// Validate validates the entire configuration.
func (c *Config) Validate() error {
	if err := c.Keyring.Validate(); err != nil {
		return fmt.Errorf("invalid keyring: %w", err)
	}
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("invalid log: %w", err)
	}
	if err := c.Query.Validate(); err != nil {
		return fmt.Errorf("invalid query: %w", err)
	}
	if err := c.RPC.Validate(); err != nil {
		return fmt.Errorf("invalid rpc: %w", err)
	}
	if err := c.Tx.Validate(); err != nil {
		return fmt.Errorf("invalid tx: %w", err)
	}

	return nil
}

// SetForFlags adds configuration flags to the specified FlagSet.
func (c *Config) SetForFlags(f *pflag.FlagSet) {
	c.Keyring.SetForFlags(f)
	c.Log.SetForFlags(f)
	c.Query.SetForFlags(f)
	c.RPC.SetForFlags(f)
	c.Tx.SetForFlags(f)
}

// DefaultConfig returns a configuration instance with default values.
func DefaultConfig() *Config {
	return &Config{
		Keyring: DefaultKeyringConfig(),
		Log:     DefaultLogConfig(),
		Query:   DefaultQueryConfig(),
		RPC:     DefaultRPCConfig(),
		Tx:      DefaultTxConfig(),
	}
}
