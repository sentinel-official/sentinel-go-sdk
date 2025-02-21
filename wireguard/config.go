package wireguard

import (
	"embed"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Embed the template files for WireGuard configurations.
//
//go:embed *.tmpl
var fs embed.FS

// PeerClientConfig represents the configuration for a single WireGuard peer.
type PeerClientConfig struct {
	AllowedAddrs        []string `mapstructure:"allowed_addrs"`        // AllowedAddrs defines the IP ranges (CIDR notation) that are allowed through this peer.
	Endpoint            string   `mapstructure:"endpoint"`             // Endpoint specifies the remote address and port of the peer.
	PersistentKeepalive uint     `mapstructure:"persistent_keepalive"` // PersistentKeepalive defines the interval (in seconds).
	PublicKey           string   `mapstructure:"public_key"`           // PublicKey is the WireGuard public key for this peer.
}

// Validate checks if the PeerClientConfig fields are correctly formatted and returns an error if any validation fails.
func (p *PeerClientConfig) Validate() error {
	// Validate AllowedAddrs (must be in CIDR notation)
	for _, ip := range p.AllowedAddrs {
		if _, err := netip.ParsePrefix(ip); err != nil {
			return fmt.Errorf("failed to parse allowed addr: %w", err)
		}
	}

	// Validate Endpoint (must be in "IP:Port" or "hostname:Port" format)
	parts := strings.Split(p.Endpoint, ":")
	if len(parts) != 2 {
		return errors.New("invalid endpoint format")
	}

	host, port := parts[0], parts[1]

	// Validate host (either an IP address or a valid hostname)
	if host == "" {
		return errors.New("host cannot be empty")
	}

	// Validate port number
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("failed to parse port: %w", err)
	}
	if portNum < 1 || portNum > 65535 {
		return errors.New("invalid port")
	}

	// Validate PublicKey
	if p.PublicKey == "" {
		return errors.New("public_key cannot be empty")
	}

	return nil
}

// ClientConfig represents the WireGuard client configuration.
type ClientConfig struct {
	Addrs      []string            `mapstructure:"addrs"`       // Addrs contains the client’s IPv4 and/or IPv6 addresses in CIDR notation.
	DNSAddrs   []string            `mapstructure:"dns_addrs"`   // DNSAddrs is a list of DNS servers to be used by the client.
	MTU        uint16              `mapstructure:"mtu"`         // MTU sets the maximum transmission unit size.
	Peers      []*PeerClientConfig `mapstructure:"peers"`       // Peers is a list of peer configurations that the client can connect to.
	Port       uint16              `mapstructure:"port"`        // Port specifies the WireGuard listening port for the client.
	PrivateKey string              `mapstructure:"private_key"` // PrivateKey holds the WireGuard private key for this client.
}

// Validate checks that the ClientConfig fields have valid values.
func (c *ClientConfig) Validate() error {
	// Validate Addrs (at lease one addr must be provided)
	if len(c.Addrs) == 0 {
		return errors.New("addrs cannot be empty")
	}

	// Validate DNSAddrs (must be valid IPs)
	for _, addr := range c.Addrs {
		if _, err := netip.ParsePrefix(addr); err != nil {
			return fmt.Errorf("invalid addr: %w", err)
		}
	}

	// Validate DNS servers (must be valid IPs)
	for _, addr := range c.DNSAddrs {
		if net.ParseIP(addr) == nil {
			return errors.New("invalid dns addr")
		}
	}

	// Validate Peers (at least one peer is required)
	if len(c.Peers) == 0 {
		return errors.New("peers cannot be empty")
	}

	for _, peer := range c.Peers {
		if err := peer.Validate(); err != nil {
			return fmt.Errorf("invalid peer config: %w", err)
		}
	}

	// Ensure Port is not empty and validate it.
	if c.Port == 0 {
		return errors.New("port cannot be empty")
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

	// Change file permissions to read and write for the owner only.
	if err := os.Chmod(name, 0600); err != nil {
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

	return strings.Join(addrs, ",")
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

	// Change file permissions to read and write for the owner only.
	if err := os.Chmod(name, 0600); err != nil {
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
