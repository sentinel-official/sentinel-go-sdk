package config

import (
	"errors"

	"github.com/spf13/pflag"
)

// LogConfig defines the configuration for logging.
type LogConfig struct {
	Format string `mapstructure:"format"` // Format of the log output (e.g., "json" or "text").
	Level  string `mapstructure:"level"`  // Logging level (e.g., "debug", "info", "warn", "error").
}

// GetFormat returns the log format.
func (c *LogConfig) GetFormat() string {
	return c.Format
}

// GetLevel returns the log level.
func (c *LogConfig) GetLevel() string {
	return c.Level
}

// Validate ensures the log configuration has valid format and level.
func (c *LogConfig) Validate() error {
	// Check if the format is valid.
	validFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validFormats[c.Format] {
		return errors.New("format must be one of: json, text")
	}

	// Check if the level is valid.
	validLevels := map[string]bool{
		"debug": true,
		"error": true,
		"info":  true,
		"warn":  true,
	}
	if !validLevels[c.Level] {
		return errors.New("level must be one of: debug, error, info, warn")
	}

	return nil
}

// SetForFlags adds log configuration flags to the specified FlagSet.
func (c *LogConfig) SetForFlags(f *pflag.FlagSet) {
	f.StringVar(&c.Format, "log.format", c.Format, "format of the log output (json or text)")
	f.StringVar(&c.Level, "log.level", c.Level, "log level for output (debug, error, info, warn)")
}

// DefaultLogConfig creates a LogConfig with default values.
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Format: "text",
		Level:  "info",
	}
}
