package wireguard

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// ServerConfig represents the WireGuard server configuration.
type ServerConfig struct {
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

	return v.InFrom
}

// OutPort returns the outbound port as a uint16.
func (c *ServerConfig) OutPort() uint16 {
	v, err := netip.NewPortFromString(c.Port)
	if err != nil {
		panic(err)
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

// Validate checks that the ServerConfig fields have valid values.
func (c *ServerConfig) Validate() error {
	// Ensure at least one of IPv4Addr or IPv6Addr is provided.
	if c.IPv4Addr == "" && c.IPv6Addr == "" {
		return errors.New("either ipv4_addr or ipv6_addr is required")
	}

	// Validate IPv4Addr if provided.
	if c.IPv4Addr != "" {
		if _, err := netip.NewPrefix(c.IPv4Addr); err != nil {
			return fmt.Errorf("invalid ipv4_addr: %w", err)
		}
	}

	// Validate IPv6Addr if provided.
	if c.IPv6Addr != "" {
		if _, err := netip.NewPrefix(c.IPv6Addr); err != nil {
			return fmt.Errorf("invalid ipv6_addr: %w", err)
		}
	}

	// Ensure OutInterface is not empty.
	if c.OutInterface == "" {
		return errors.New("out_interface cannot be empty")
	}

	// Ensure Port is not empty and validate it.
	if c.Port == "" {
		return errors.New("port cannot be empty")
	}
	if _, err := netip.NewPortFromString(c.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	// Ensure PrivateKey is not empty and validate it.
	if c.PrivateKey == "" {
		return errors.New("private_key cannot be empty")
	}
	if _, err := NewKeyFromString(c.PrivateKey); err != nil {
		return fmt.Errorf("invalid private_key: %w", err)
	}

	return nil
}

// WriteServiceConfig generates the service-level configuration file using the service template.
func (c *ServerConfig) WriteServiceConfig(filename string) error {
	// Load the service template from the embedded or filesystem path.
	text, err := fs.ReadFile("server.conf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read service template: %w", err)
	}

	// Render the template with ServerConfig data and write the result to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, filename); err != nil {
		return fmt.Errorf("failed to write rendered service config to file: %w", err)
	}

	// Restrict file permissions to owner read/write only.
	if err := os.Chmod(filename, 0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// WriteAppConfig generates the application-level configuration file using the main config template.
func (c *ServerConfig) WriteAppConfig(filename string) error {
	// Load the application config template from the embedded or filesystem path.
	text, err := fs.ReadFile("server_config.toml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read application config template: %w", err)
	}

	// Render the template with ServerConfig data and write the result to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, filename); err != nil {
		return fmt.Errorf("failed to write rendered application config to file: %w", err)
	}

	// Restrict file permissions to owner read/write only.
	if err := os.Chmod(filename, 0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// AddrPoolSet returns all address pools (IPv4 and IPv6).
func (c *ServerConfig) AddrPoolSet() (*netip.AddrPoolSet, error) {
	pool, err := netip.NewAddrPoolSet(c.IPv4Addr, c.IPv6Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create addr pool set: %w", err)
	}

	return pool, nil
}

// SetForFlags adds server configuration flags to the specified FlagSet.
func (c *ServerConfig) SetForFlags(_ *pflag.FlagSet) {}

// DefaultServerConfig creates a default ServerConfig with default values.
func DefaultServerConfig() *ServerConfig {
	pk, err := NewPrivateKey()
	if err != nil {
		panic(err)
	}

	return &ServerConfig{
		IPv4Addr:     fmt.Sprintf("10.%d.%d.1/24", rand.Intn(256), rand.Intn(256)),
		IPv6Addr:     "",
		OutInterface: "eth0",
		Port:         fmt.Sprintf("%d", utils.RandomPort()),
		PrivateKey:   pk.String(),
	}
}
