package wireguard

import (
	"embed"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/spf13/pflag"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Embed the template files for WireGuard configurations.
//
//go:embed *.tmpl
var fs embed.FS

// ClientConfig represents the WireGuard client configuration.
type ClientConfig struct{}

// Validate verifies the ClientConfig. (No-op for now.)
func (c *ClientConfig) Validate() error {
	return nil
}

// WriteToFile writes the client configuration template to a file.
func (c *ClientConfig) WriteToFile(name string) error {
	// Read the client configuration template file.
	text, err := fs.ReadFile("client.conf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Execute the template and write it to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, name); err != nil {
		return fmt.Errorf("failed to execute template to file: %w", err)
	}

	// Change file permissions to read-only.
	if err := os.Chmod(name, 0400); err != nil {
		return fmt.Errorf("failed to change file permissions: %w", err)
	}

	return nil
}

// ServerConfig represents the WireGuard server configuration.
type ServerConfig struct {
	InInterface  string `mapstructure:"in_interface"`  // InInterface specifies the inbound interface.
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

	return strings.Join(addrs, ", ")
}

// InPort returns the inbound port as a uint16.
func (c *ServerConfig) InPort() uint16 {
	v, err := types.NewPortFromString(c.Port)
	if err != nil {
		panic(err)
	}

	return v.InFrom
}

// OutPort returns the outbound port as a uint16.
func (c *ServerConfig) OutPort() uint16 {
	v, err := types.NewPortFromString(c.Port)
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
	// Ensure InInterface is not empty.
	if c.InInterface == "" {
		return errors.New("in_interface cannot be empty")
	}

	// Ensure at least one of IPv4Addr or IPv6Addr is provided.
	if c.IPv4Addr == "" && c.IPv6Addr == "" {
		return errors.New("either ipv4_addr or ipv6_addr is required")
	}

	// Validate IPv4Addr if provided.
	if c.IPv4Addr != "" {
		if _, err := types.NewNetPrefixFromString(c.IPv4Addr); err != nil {
			return fmt.Errorf("invalid ipv4_addr: %w", err)
		}
	}

	// Validate IPv6Addr if provided.
	if c.IPv6Addr != "" {
		if _, err := types.NewNetPrefixFromString(c.IPv6Addr); err != nil {
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
	if _, err := types.NewPortFromString(c.Port); err != nil {
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

// WriteToFile writes the server configuration template to a file.
func (c *ServerConfig) WriteToFile(name string) error {
	// Read the server configuration template file.
	text, err := fs.ReadFile("server.conf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Execute the template and write it to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, name); err != nil {
		return fmt.Errorf("failed to execute template to file: %w", err)
	}

	// Change file permissions to read-only.
	if err := os.Chmod(name, 0400); err != nil {
		return fmt.Errorf("failed to change file permissions: %w", err)
	}

	return nil
}

// IPv4Pool returns the IPv4 address pool.
func (c *ServerConfig) IPv4Pool() (*types.IPPool, error) {
	pool, err := types.NewIPPoolFromString(c.IPv4Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get ip pool: %w", err)
	}

	return pool, nil
}

// IPv6Pool returns the IPv6 address pool.
func (c *ServerConfig) IPv6Pool() (*types.IPPool, error) {
	pool, err := types.NewIPPoolFromString(c.IPv6Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get ip pool: %w", err)
	}

	return pool, nil
}

// IPPools returns all address pools (IPv4 and IPv6).
func (c *ServerConfig) IPPools() ([]*types.IPPool, error) {
	var pools []*types.IPPool

	// Add the IPv4 pool if IPv4Addr is provided.
	if c.IPv4Addr != "" {
		pool, err := c.IPv4Pool()
		if err != nil {
			return nil, fmt.Errorf("failed to get ipv4 pool: %w", err)
		}

		// Append the IPv4 pool to the list.
		pools = append(pools, pool)
	}

	// Add the IPv6 pool if IPv6Addr is provided.
	if c.IPv6Addr != "" {
		pool, err := c.IPv6Pool()
		if err != nil {
			return nil, fmt.Errorf("failed to get ipv6 pool: %w", err)
		}

		// Append the IPv6 pool to the list.
		pools = append(pools, pool)
	}

	return pools, nil
}

// SetForFlags adds server configuration flags to the specified FlagSet.
func (c *ServerConfig) SetForFlags(_ *pflag.FlagSet) {}

// DefaultServerConfig creates a default ServerConfig with randomized values.
func DefaultServerConfig() *ServerConfig {
	pk, err := NewPrivateKey()
	if err != nil {
		panic(err)
	}

	return &ServerConfig{
		InInterface:  "wg0",
		IPv4Addr:     fmt.Sprintf("10.%d.%d.1/24", rand.Intn(256), rand.Intn(256)),
		IPv6Addr:     "",
		OutInterface: "eth0",
		Port:         fmt.Sprintf("%d", utils.RandomPort()),
		PrivateKey:   pk.String(),
	}
}
