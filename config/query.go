package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

// QueryConfig defines the configuration for query operations.
type QueryConfig struct {
	Prove      bool   `mapstructure:"prove"`       // Prove indicates whether to include proof in query results.
	Retries    uint   `mapstructure:"retries"`     // Retries is the number of retries for the query.
	RetryDelay string `mapstructure:"retry_delay"` // RetryDelay is the delay between query retries.
}

// GetProve returns whether to include proof in query results.
func (c *QueryConfig) GetProve() bool {
	return c.Prove
}

// GetRetries returns the maximum number of retries for the query.
func (c *QueryConfig) GetRetries() uint {
	return c.Retries
}

// GetRetryDelay returns the delay between retries for the query as a time.Duration.
func (c *QueryConfig) GetRetryDelay() time.Duration {
	v, err := time.ParseDuration(c.RetryDelay)
	if err != nil {
		panic(err)
	}

	return v
}

// Validate checks the Query configuration for validity.
func (c *QueryConfig) Validate() error {
	// Ensure Retries is non-zero.
	if c.Retries == 0 {
		return errors.New("retries cannot be zero")
	}

	// Ensure RetryDelay is a valid time.Duration string.
	if _, err := time.ParseDuration(c.RetryDelay); err != nil {
		return fmt.Errorf("invalid retry_delay: %w", err)
	}

	return nil
}

// SetForFlags adds query configuration flags to the specified FlagSet.
func (c *QueryConfig) SetForFlags(f *pflag.FlagSet) {
	f.BoolVar(&c.Prove, "query.prove", c.Prove, "include proof in query results")
	f.UintVar(&c.Retries, "query.retries", c.Retries, "number of retries for the query")
	f.StringVar(&c.RetryDelay, "query.retry-delay", c.RetryDelay, "delay between query retries (e.g., 2s, 500ms)")
}

// DefaultQueryConfig creates a QueryConfig with default values.
func DefaultQueryConfig() *QueryConfig {
	return &QueryConfig{
		Prove:      false,
		Retries:    5,
		RetryDelay: "2s",
	}
}
