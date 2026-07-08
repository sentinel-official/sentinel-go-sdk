package wireguard

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// ServerConfig represents the WireGuard server configuration.
type ServerConfig struct {
	viper *viper.Viper `mapstructure:"-"`

	IPv4Addr     string `mapstructure:"ipv4_addr"`     // IPv4Addr is the IPv4 address with CIDR notation.
	IPv6Addr     string `mapstructure:"ipv6_addr"`     // IPv6Addr is the IPv6 address with CIDR notation.
	OutInterface string `mapstructure:"out_interface"` // OutInterface specifies the outbound interface.
	Port         string `mapstructure:"port"`          // Port specifies the WireGuard listening port.
	PrivateKey   string `mapstructure:"private_key"`   // PrivateKey is the WireGuard private key.
}

// Address returns the combined IPv4 and IPv6 addresses, separated by a comma.
func (c *ServerConfig) Address() string {
	var addrs []string
	if c.IPv4Addr != "" {
		addrs = append(addrs, c.IPv4Addr)
	}

	if c.IPv6Addr != "" {
		addrs = append(addrs, c.IPv6Addr)
	}

	return strings.Join(addrs, ",")
}

// InPort returns the inbound port as a uint16.
func (c *ServerConfig) InPort() uint16 {
	v, err := netip.NewPortFromString(c.Port)
	if err != nil {
		panic(err)
	}

	if v == nil {
		panic(errors.New("nil port"))
	}

	return v.InFrom
}

// OutPort returns the outbound port as a uint16.
func (c *ServerConfig) OutPort() uint16 {
	v, err := netip.NewPortFromString(c.Port)
	if err != nil {
		panic(err)
	}

	if v == nil {
		panic(errors.New("nil port"))
	}

	return v.OutFrom
}

// PublicKey returns the public key derived from the private key.
func (c *ServerConfig) PublicKey() *Key {
	pk, err := NewKeyFromString(c.PrivateKey)
	if err != nil {
		panic(err)
	}

	return pk.Public()
}

// AddrPoolSet returns all address pools (IPv4 and IPv6).
func (c *ServerConfig) AddrPoolSet() (*netip.AddrPoolSet, error) {
	var addrs []string
	if c.IPv4Addr != "" {
		addrs = append(addrs, c.IPv4Addr)
	}

	if c.IPv6Addr != "" {
		addrs = append(addrs, c.IPv6Addr)
	}

	pool, err := netip.NewAddrPoolSet(addrs...)
	if err != nil {
		return nil, fmt.Errorf("creating addr pool set: %w", err)
	}

	return pool, nil
}

// Validate checks that the ServerConfig fields have valid values.
func (c *ServerConfig) Validate() error {
	// Ensure at least one of IPv4Addr or IPv6Addr is provided.
	if c.IPv4Addr == "" && c.IPv6Addr == "" {
		return errors.New("ipv4_addr and ipv6_addr are empty")
	}

	// Validate IPv4Addr if provided.
	if c.IPv4Addr != "" {
		if _, err := netip.NewPrefix(c.IPv4Addr); err != nil {
			return fmt.Errorf("parsing ipv4_addr %q: %w", c.IPv4Addr, err)
		}
	}

	// Validate IPv6Addr if provided.
	if c.IPv6Addr != "" {
		if _, err := netip.NewPrefix(c.IPv6Addr); err != nil {
			return fmt.Errorf("parsing ipv6_addr %q: %w", c.IPv4Addr, err)
		}
	}

	// Ensure OutInterface is a valid interface name (rejects shell metacharacters).
	if !utils.IsValidInterfaceName(c.OutInterface) {
		return fmt.Errorf("invalid out_interface %q", c.OutInterface)
	}

	// Ensure Port is not empty and validate it.
	if c.Port == "" {
		return errors.New("port is empty")
	}

	if _, err := netip.NewPortFromString(c.Port); err != nil {
		return fmt.Errorf("parsing port %q: %w", c.Port, err)
	}

	// Ensure PrivateKey is not empty and validate it.
	if c.PrivateKey == "" {
		return errors.New("private_key is empty")
	}

	if _, err := NewKeyFromString(c.PrivateKey); err != nil {
		return fmt.Errorf("parsing private_key %q: %w", c.PrivateKey, err)
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
	text, err := fs.ReadFile("server.conf.tmpl")
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

// DefaultServerConfig creates a default ServerConfig with default values.
func DefaultServerConfig() *ServerConfig {
	pk, err := NewPrivateKey()
	if err != nil {
		panic(err)
	}

	return &ServerConfig{
		IPv4Addr:     fmt.Sprintf("10.%d.%d.1/24", rand.IntN(256), rand.IntN(256)),
		IPv6Addr:     "",
		OutInterface: "eth0",
		Port:         strconv.FormatUint(uint64(utils.RandomPort()), 10),
		PrivateKey:   pk.String(),
	}
}
