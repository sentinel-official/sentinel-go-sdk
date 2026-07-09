package hysteria2

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/v2/utils"
)

// ServerConfig represents the Hysteria2 server configuration options.
type ServerConfig struct {
	viper *viper.Viper `mapstructure:"-"`

	Port          uint16 `mapstructure:"port"`           // Port defines the server listening port.
	TLSCertFile   string `mapstructure:"-"`              // TLSCertFile is the path to the TLS certificate file.
	TLSKeyFile    string `mapstructure:"-"`              // TLSKeyFile is the path to the TLS private key file.
	ObfsPassword  string `mapstructure:"obfs_password"`  // ObfsPassword is the Salamander obfuscation password (empty disables obfs).
	MasqueradeURL string `mapstructure:"masquerade_url"` // MasqueradeURL, when set, reverse-proxies unauthenticated probes to it; empty serves a self-contained 404.
	AuthPort      uint16 `mapstructure:"auth_port"`      // AuthPort is the loopback port for the SDK-hosted HTTP auth backend.
	StatsPort     uint16 `mapstructure:"stats_port"`     // StatsPort is the loopback port for Hysteria2's Traffic Stats API.
	StatsSecret   string `mapstructure:"stats_secret"`   // StatsSecret is the authorization secret for the Traffic Stats API.
}

// Validate validates the ServerConfig fields.
func (c *ServerConfig) Validate() error {
	// Ensure Port is not zero.
	if c.Port == 0 {
		return errors.New("port is zero")
	}

	// Validate TLS certificate file is specified.
	if c.TLSCertFile == "" {
		return errors.New("tls_cert_file is empty")
	}

	// Validate TLS key file is specified.
	if c.TLSKeyFile == "" {
		return errors.New("tls_key_file is empty")
	}

	// Ensure AuthPort is not zero.
	if c.AuthPort == 0 {
		return errors.New("auth_port is zero")
	}

	// Ensure StatsPort is not zero.
	if c.StatsPort == 0 {
		return errors.New("stats_port is zero")
	}

	// Ensure StatsSecret is not empty.
	if c.StatsSecret == "" {
		return errors.New("stats_secret is empty")
	}

	// Validate MasqueradeURL when set.
	if c.MasqueradeURL != "" {
		u, err := url.Parse(c.MasqueradeURL)
		if err != nil {
			return fmt.Errorf("parsing masquerade_url %q: %w", c.MasqueradeURL, err)
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("invalid masquerade_url scheme %q", u.Scheme)
		}
	}

	return nil
}

// ReadAppConfig reads the application configuration from the specified file.
func (c *ServerConfig) ReadAppConfig(file string) error {
	// Initialize Viper instance if it hasn't been already.
	if c.viper == nil {
		c.viper = viper.New()
	}

	// Set the path to the config file.
	c.viper.SetConfigFile(file)

	// Read the configuration file from disk.
	if err := c.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("reading config file %q: %w", file, err)
	}

	// Unmarshal the config data into the ServerConfig struct.
	if err := c.viper.Unmarshal(c); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	return nil
}

// WriteAppConfig generates the application-level configuration file using the main config template.
func (c *ServerConfig) WriteAppConfig(file string) error {
	// Load the application template from the embedded filesystem.
	text, err := fs.ReadFile("server_config.toml.tmpl")
	if err != nil {
		return fmt.Errorf("reading config template: %w", err)
	}

	// Render the template with ServerConfig data and write the result to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, file); err != nil {
		return fmt.Errorf("writing rendered config file %q: %w", file, err)
	}

	// Restrict file permissions to owner read/write only.
	if err := os.Chmod(file, 0600); err != nil {
		return fmt.Errorf("setting file permissions: %w", err)
	}

	return nil
}

// WriteServiceConfig generates the service-level configuration file using the service template.
func (c *ServerConfig) WriteServiceConfig(file string) error {
	// Load the service template from the embedded filesystem.
	text, err := fs.ReadFile("server.yaml.tmpl")
	if err != nil {
		return fmt.Errorf("reading config template: %w", err)
	}

	// Render the template with ServerConfig data and write the result to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, file); err != nil {
		return fmt.Errorf("writing rendered config file %q: %w", file, err)
	}

	// Restrict file permissions to owner read/write only.
	if err := os.Chmod(file, 0600); err != nil {
		return fmt.Errorf("setting file permissions: %w", err)
	}

	return nil
}

// SetForFlags adds server configuration flags to the specified FlagSet.
func (c *ServerConfig) SetForFlags(_ *pflag.FlagSet, _ string) {}

// DefaultServerConfig creates a default ServerConfig with predefined values.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:          utils.RandomPort(),
		ObfsPassword:  uuid.NewString(),
		MasqueradeURL: "",
		AuthPort:      utils.RandomPort(),
		StatsPort:     utils.RandomPort(),
		StatsSecret:   uuid.NewString(),
	}
}
