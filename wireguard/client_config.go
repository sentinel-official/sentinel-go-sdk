package wireguard

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"

	"github.com/spf13/pflag"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// PeerClientConfig represents the configuration for a single WireGuard peer.
type PeerClientConfig struct {
	Addr                string   `mapstructure:"addr"`                 // Addr specifies the IP address or hostname of the peer.
	AllowAddrs          []string `mapstructure:"allow_addrs"`          // AllowAddrs defines the IP ranges (CIDR notation) that are allowed through this peer.
	PersistentKeepalive uint     `mapstructure:"persistent_keepalive"` // PersistentKeepalive defines the interval (in seconds).
	Port                uint16   `mapstructure:"port"`                 // Port is the listening port of the peer.
	PublicKey           string   `mapstructure:"public_key"`           // PublicKey is the WireGuard public key for this peer.
}

// Endpoint returns the full network endpoint for the WireGuard peer.
func (c *PeerClientConfig) Endpoint() string {
	return net.JoinHostPort(c.Addr, fmt.Sprintf("%d", c.Port))
}

// Validate checks if the PeerClientConfig fields are correctly formatted and returns an error if any validation fails.
func (c *PeerClientConfig) Validate() error {
	// Ensure that Addr is not empty.
	if c.Addr == "" {
		return errors.New("addr cannot be empty")
	}

	// Validate AllowAddrs (must be in CIDR notation)
	for _, addr := range c.AllowAddrs {
		if _, err := netip.ParsePrefix(addr); err != nil {
			return fmt.Errorf("failed to parse addr: %w", err)
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

	// Validate PublicKey (must be non-empty and a valid WireGuard public key).
	if c.PublicKey == "" {
		return errors.New("public_key cannot be empty")
	}
	if _, err := NewKeyFromString(c.PublicKey); err != nil {
		return fmt.Errorf("invalid public_key: %w", err)
	}

	return nil
}

// DefaultPeerClientConfig creates a default PeerClientConfig with default values.
func DefaultPeerClientConfig() *PeerClientConfig {
	return &PeerClientConfig{
		Addr:                "",
		AllowAddrs:          []string{"0.0.0.0/0", "::/0"},
		PersistentKeepalive: 30,
		Port:                0,
		PublicKey:           "",
	}
}

// ClientConfig represents the WireGuard client configuration.
type ClientConfig struct {
	Addrs        []string          `mapstructure:"addrs"`         // Addrs contains the client’s IPv4 and/or IPv6 addresses in CIDR notation.
	DNSAddrs     []string          `mapstructure:"dns_addrs"`     // DNSAddrs is a list of DNS servers to be used by the client.
	ExcludeAddrs []string          `mapstructure:"exclude_addrs"` // ExcludeAddrs defines IP ranges that should not use the VPN tunnel.
	MTU          uint16            `mapstructure:"mtu"`           // MTU sets the maximum transmission unit size.
	Name         string            `mapstructure:"name"`          // Name is the name of the WireGuard interface.
	Peer         *PeerClientConfig `mapstructure:"peers"`         // Peer is a peer configurations that the client can connect to.
	Port         uint16            `mapstructure:"port"`          // Port specifies the WireGuard listening port for the client.
	PrivateKey   string            `mapstructure:"private_key"`   // PrivateKey holds the WireGuard private key for this client.
}

// GetAddrs returns the list of addresses (Addrs) as netip.Prefixes.
func (c *ClientConfig) GetAddrs() []netip.Prefix {
	var addrs []netip.Prefix
	for _, addr := range c.Addrs {
		addr, err := netip.ParsePrefix(addr)
		if err != nil {
			panic(err)
		}

		addrs = append(addrs, addr)
	}

	return addrs
}

// GetExcludeAddrs returns the list of exclude addresses (ExcludeAddrs) as netip.Prefixes.
func (c *ClientConfig) GetExcludeAddrs() []netip.Prefix {
	var addrs []netip.Prefix
	for _, addr := range c.ExcludeAddrs {
		addr, err := netip.ParsePrefix(addr)
		if err != nil {
			panic(err)
		}

		addrs = append(addrs, addr)
	}

	return addrs
}

// GetPrivateKey returns the private key associated with the client configuration.
func (c *ClientConfig) GetPrivateKey() *Key {
	key, err := NewKeyFromString(c.PrivateKey)
	if err != nil {
		panic(err)
	}

	return key
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

	// Validate Peer (must be non-empty and a valid PeerClientConfig).
	if c.Peer == nil {
		return errors.New("peer cannot be empty")
	}
	if err := c.Peer.Validate(); err != nil {
		return fmt.Errorf("invalid peer config: %w", err)
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

// WriteServiceConfig generates the service-level configuration file using the service template.
func (c *ClientConfig) WriteServiceConfig(filename string) error {
	// Load the service template from the embedded or filesystem path.
	text, err := fs.ReadFile("client.conf.tmpl")
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
func (c *ClientConfig) WriteAppConfig(filename string) error {
	// Load the application config template from the embedded or filesystem path.
	text, err := fs.ReadFile("client_config.toml.tmpl")
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

// SetForFlags adds client configuration flags to the specified FlagSet.
func (c *ClientConfig) SetForFlags(f *pflag.FlagSet) {
	f.StringArrayVar(&c.DNSAddrs, "wg.dns-addrs", c.DNSAddrs, "dns servers to use while connected to the vpn")
	f.StringArrayVar(&c.ExcludeAddrs, "wg.exclude-addrs", c.ExcludeAddrs, "exclude ip addresses/subnets from the wireguard tunnel")
	f.Uint16Var(&c.MTU, "wg.mtu", c.MTU, "maximum transmission unit size for the wireguard interface")
	f.StringVar(&c.Name, "wg.name", c.Name, "name of the wireguard network interface")
	f.StringArrayVar(&c.Peer.AllowAddrs, "wg.peer.allow-addrs", c.Peer.AllowAddrs, "list of allowed ip addresses to route through wireguard peer")
	f.UintVar(&c.Peer.PersistentKeepalive, "wg.peer.persistent-keepalive", c.Peer.PersistentKeepalive, "interval for keepalive packets to maintain connection")
	f.Uint16Var(&c.Port, "wg.port", c.Port, "port number for the wireguard interface")
}

// DefaultClientConfig creates a default ClientConfig with default values.
func DefaultClientConfig() *ClientConfig {
	privateKey, err := NewPrivateKey()
	if err != nil {
		panic(err)
	}

	return &ClientConfig{
		Addrs:        nil,
		DNSAddrs:     []string{"208.67.222.222", "208.67.220.220", "2620:119:35::35", "2620:119:53::53"},
		ExcludeAddrs: []string{"127.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12", "10.0.0.0/8", "::1/128", "fe80::/10", "fd00::/8"},
		MTU:          1420,
		Name:         "wg0",
		Peer:         DefaultPeerClientConfig(),
		Port:         utils.RandomPort(),
		PrivateKey:   privateKey.String(),
	}
}
