package hysteria2

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// ClientConfig represents the Hysteria2 client configuration options.
type ClientConfig struct {
	viper *viper.Viper `mapstructure:"-"`

	ServerAddr   string `mapstructure:"server_addr"`   // ServerAddr is the server address (host:port).
	Auth         string `mapstructure:"auth"`          // Auth is the UUID used to authenticate with the server.
	TLSPin       string `mapstructure:"tls_pin"`       // TLSPin is the SHA-256 certificate fingerprint (hex-encoded).
	TUNIface     string `mapstructure:"tun_iface"`     // TUNIface is the name of the TUN network interface.
	ObfsPassword string `mapstructure:"obfs_password"` // ObfsPassword is the Salamander obfuscation password (empty disables obfs).
}

// Validate validates the ClientConfig fields.
func (c *ClientConfig) Validate() error {
	// Ensure ServerAddr is not empty.
	if c.ServerAddr == "" {
		return errors.New("server_addr is empty")
	}

	// Ensure Auth is not empty.
	if c.Auth == "" {
		return errors.New("auth is empty")
	}

	// Ensure TLSPin is not empty.
	if c.TLSPin == "" {
		return errors.New("tls_pin is empty")
	}

	// Ensure TUNIface is not empty.
	if c.TUNIface == "" {
		return errors.New("tun_iface is empty")
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
	text, err := fs.ReadFile("client.yaml.tmpl")
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

// SetForFlags adds client configuration flags to the specified FlagSet.
func (c *ClientConfig) SetForFlags(fs *pflag.FlagSet, prefix string) {
	if prefix != "" {
		prefix += "."
	}

	fs.StringVar(&c.TUNIface, prefix+"tun_iface", c.TUNIface, "name of the TUN network interface for the hysteria2 client")

	// Initialize Viper if it hasn't been already.
	if c.viper == nil {
		c.viper = viper.New()
	}

	// Bind all added flags.
	r := strings.NewReplacer("-", "_")

	fs.VisitAll(func(f *pflag.Flag) {
		if strings.HasPrefix(f.Name, prefix) {
			_ = c.viper.BindPFlag(strings.TrimPrefix(r.Replace(f.Name), prefix), f)
		}
	})
}

// DefaultClientConfig creates a default ClientConfig with predefined values.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerAddr:   "",
		Auth:         NewUUID(),
		TLSPin:       "",
		TUNIface:     "hyst0",
		ObfsPassword: "",
	}
}
