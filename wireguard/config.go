package wireguard

import (
	"embed"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
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

// PeerClientConfig represents the configuration for a single WireGuard peer.
type PeerClientConfig struct {
	Addr                string   `mapstructure:"addr"`                 // Addr specifies the IP address or hostname of the peer.
	AllowAddrs          []string `mapstructure:"allow_addrs"`          // AllowAddrs defines the IP ranges (CIDR notation) that are allowed through this peer.
	PersistentKeepalive uint     `mapstructure:"persistent_keepalive"` // PersistentKeepalive defines the interval (in seconds).
	Port                uint16   `mapstructure:"port"`                 // Port is the listening port of the peer.
	PublicKey           string   `mapstructure:"public_key"`           // PublicKey is the WireGuard public key for this peer.
}

func (c *PeerClientConfig) Endpoint() string {
	return fmt.Sprintf("%s:%d", c.Addr, c.Port)
}

// Validate checks if the PeerClientConfig fields are correctly formatted and returns an error if any validation fails.
func (c *PeerClientConfig) Validate() error {
	// Ensure that Addr is not empty.
	if c.Addr == "" {
		return errors.New("addr cannot be empty")
	}

	// Validate AllowAddrs (must be in CIDR notation)
	for _, ip := range c.AllowAddrs {
		if _, err := netip.ParsePrefix(ip); err != nil {
			return fmt.Errorf("failed to parse allow addr: %w", err)
		}
	}

	// Ensure PersistentKeepalive is set to a valid non-zero value.
	if c.PersistentKeepalive == 0 {
		return errors.New("persistent_keepalive cannot be empty")
	}

	// Validate Port (must be a non-zero value).
	if c.Port == 0 {
		return errors.New("port cannot be empty")
	}

	// Ensure PublicKey is not empty.
	if c.PublicKey == "" {
		return errors.New("public_key cannot be empty")
	}

	return nil
}

// ClientConfig represents the WireGuard client configuration.
type ClientConfig struct {
	Addrs        []string            `mapstructure:"addrs"`         // Addrs contains the client’s IPv4 and/or IPv6 addresses in CIDR notation.
	DNSAddrs     []string            `mapstructure:"dns_addrs"`     // DNSAddrs is a list of DNS servers to be used by the client.
	ExcludeAddrs []string            `mapstructure:"exclude_addrs"` // ExcludeAddrs defines IP ranges that should not use the VPN tunnel.
	MTU          uint16              `mapstructure:"mtu"`           // MTU sets the maximum transmission unit size.
	Name         string              `mapstructure:"name"`          // Name is the name of the WireGuard interface.
	Peers        []*PeerClientConfig `mapstructure:"peers"`         // Peers is a list of peer configurations that the client can connect to.
	Port         uint16              `mapstructure:"port"`          // Port specifies the WireGuard listening port for the client.
	PrivateKey   string              `mapstructure:"private_key"`   // PrivateKey holds the WireGuard private key for this client.
}

// GetAddrs returns the list of addresses (Addrs) as netip.Prefixes.
func (c *ClientConfig) GetAddrs() []netip.Prefix {
	var addrs []netip.Prefix
	for _, addr := range c.Addrs {
		prefix, err := netip.ParsePrefix(addr)
		if err != nil {
			panic(fmt.Errorf("failed to parse addr: %w", err))
		}

		addrs = append(addrs, prefix)
	}

	return addrs
}

// GetExcludeAddrs returns the list of exclude addresses (ExcludeAddrs) as netip.Prefixes.
func (c *ClientConfig) GetExcludeAddrs() []netip.Prefix {
	var addrs []netip.Prefix
	for _, addr := range c.ExcludeAddrs {
		prefix, err := netip.ParsePrefix(addr)
		if err != nil {
			panic(fmt.Errorf("failed to parse addr: %w", err))
		}

		addrs = append(addrs, prefix)
	}

	return addrs
}

// Validate checks that all fields in ClientConfig have valid values.
func (c *ClientConfig) Validate() error {
	// Validate Addrs (at least one address must be provided).
	if len(c.Addrs) == 0 {
		return errors.New("addrs cannot be empty")
	}

	// Validate that each address in Addrs is a valid network prefix in CIDR notation.
	for _, addr := range c.Addrs {
		if _, err := netip.ParsePrefix(addr); err != nil {
			return fmt.Errorf("invalid addr: %w", err)
		}
	}

	// Validate DNSAddrs (must be valid IP addresses).
	for _, addr := range c.DNSAddrs {
		if net.ParseIP(addr) == nil {
			return errors.New("invalid dns addr")
		}
	}

	// Validate ExcludeAddrs (if provided, each address must be a valid CIDR range).
	for _, addr := range c.ExcludeAddrs {
		if _, err := netip.ParsePrefix(addr); err != nil {
			return fmt.Errorf("failed to parse excluded addr: %w", err)
		}
	}

	// Validate MTU (must be a non-zero value).
	if c.MTU == 0 {
		return errors.New("mtu cannot be empty")
	}

	// Ensure Name is not empty.
	if c.Name == "" {
		return errors.New("name cannot be empty")
	}

	// Validate Peers (at least one peer must be configured).
	if len(c.Peers) == 0 {
		return errors.New("peers cannot be empty")
	}

	// Validate each peer configuration.
	for _, peer := range c.Peers {
		if err := peer.Validate(); err != nil {
			return fmt.Errorf("invalid peer config: %w", err)
		}
	}

	// Validate Port (must be a non-zero value).
	if c.Port == 0 {
		return errors.New("port cannot be empty")
	}

	// Validate PrivateKey (must be non-empty and a valid WireGuard private key).
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
