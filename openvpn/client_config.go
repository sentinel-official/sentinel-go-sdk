package openvpn

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Supported transport protocols.
const (
	protoTCP = "tcp"
	protoUDP = "udp"
)

// ClientConfig defines the configuration required to set up an OpenVPN client.
type ClientConfig struct {
	viper *viper.Viper `mapstructure:"-"`

	Addr     string `mapstructure:"-"` // Addr specifies the server address to connect to.
	CA       []byte `mapstructure:"-"` // CA is the Certificate Authority certificate in DER format.
	Cert     []byte `mapstructure:"-"` // Cert is the client certificate in DER format.
	Key      []byte `mapstructure:"-"` // Key is the client private key in DER format.
	PKIDir   string `mapstructure:"-"` // PKIDir is the path to the PKI directory used for certificates.
	Port     uint16 `mapstructure:"-"` // Port specifies the server port to connect to.
	Protocol string `mapstructure:"-"` // Protocol specifies the transport protocol (either "tcp" or "udp").
	TLS      []byte `mapstructure:"-"` // TLS is the static TLS key in raw bytes.
}

// Validate checks the correctness of all configuration fields.
func (c *ClientConfig) Validate() error {
	// Ensure Addr is not empty.
	if c.Addr == "" {
		return errors.New("addr is empty")
	}

	// Ensure Port is not zero.
	if c.Port == 0 {
		return errors.New("port is zero")
	}

	// Protocol must be either "tcp" or "udp"
	validProtocols := map[string]bool{
		protoTCP: true,
		protoUDP: true,
	}
	if !validProtocols[c.Protocol] {
		return fmt.Errorf("unsupported protocol %q (allowed: tcp, udp)", c.Protocol)
	}

	// CA certificate must be non-empty.
	if len(c.CA) == 0 {
		return errors.New("ca is empty")
	}

	// Client certificate must be non-empty.
	if len(c.Cert) == 0 {
		return errors.New("cert is empty")
	}

	// Client private key must be non-empty.
	if len(c.Key) == 0 {
		return errors.New("key is empty")
	}

	// PKI directory must be non-empty.
	if c.PKIDir == "" {
		return errors.New("pki_dir is empty")
	}

	// TLS static key must be non-empty.
	if len(c.TLS) == 0 {
		return errors.New("tls is empty")
	}

	return nil
}

// ReadAppConfig reads the application configuration from the specified file.
func (c *ClientConfig) ReadAppConfig(file string) error {
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

	// Unmarshal the config data into the ClientConfig struct.
	if err := c.viper.Unmarshal(c); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	return nil
}

// WriteAppConfig generates the application-level configuration file using the main config template.
func (c *ClientConfig) WriteAppConfig(file string) error {
	// Load the application template from the embedded filesystem.
	text, err := fs.ReadFile("client_config.toml.tmpl")
	if err != nil {
		return fmt.Errorf("reading config template: %w", err)
	}

	// Render the template with ClientConfig data and write the result to the specified file.
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
func (c *ClientConfig) WriteServiceConfig(file string) error {
	// Load the service template from the embedded filesystem.
	text, err := fs.ReadFile("client.conf.tmpl")
	if err != nil {
		return fmt.Errorf("reading config template: %w", err)
	}

	// Render the template with ClientConfig data and write the result to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, file); err != nil {
		return fmt.Errorf("writing rendered config file %q: %w", file, err)
	}

	// Restrict file permissions to owner read/write only.
	if err := os.Chmod(file, 0600); err != nil {
		return fmt.Errorf("setting file permissions: %w", err)
	}

	return nil
}

// SetForFlags is a placeholder method to allow binding ClientConfig to CLI flags.
func (c *ClientConfig) SetForFlags(_ *pflag.FlagSet, _ string) {}

// DefaultClientConfig returns a ClientConfig instance with zero values.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{}
}
