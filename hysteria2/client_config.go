package hysteria2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// ClientConfig represents the Hysteria2 client configuration options.
type ClientConfig struct {
	viper *viper.Viper `mapstructure:"-"`

	resolvedServerIPs []net.IP `mapstructure:"-"`

	ServerAddr   string   `mapstructure:"server_addr"`   // ServerAddr is the server address (host:port).
	Auth         string   `mapstructure:"auth"`          // Auth is the UUID used to authenticate with the server.
	TLSPin       string   `mapstructure:"tls_pin"`       // TLSPin is the SHA-256 certificate fingerprint (hex-encoded).
	TUNIface     string   `mapstructure:"tun_iface"`     // TUNIface is the name of the TUN network interface.
	Addrs        []string `mapstructure:"addrs"`         // Addrs contains the client's IPv4 and/or IPv6 addresses in CIDR notation for the TUN interface.
	RouteAddrs   []string `mapstructure:"route_addrs"`   // RouteAddrs defines the IP ranges (CIDR notation) routed through the tunnel.
	ExcludeAddrs []string `mapstructure:"exclude_addrs"` // ExcludeAddrs defines IP ranges that should not use the tunnel.
	DNSAddrs     []string `mapstructure:"dns_addrs"`     // DNSAddrs is a list of DNS servers to be used by the client.
	MTU          uint16   `mapstructure:"mtu"`           // MTU sets the maximum transmission unit size for the TUN interface.
	ObfsPassword string   `mapstructure:"obfs_password"` // ObfsPassword is the Salamander obfuscation password (empty disables obfs).
}

// ipv4Prefixes returns the subset of addrs that are IPv4 prefixes.
func ipv4Prefixes(addrs []string) []string {
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		prefix, err := netip.ParsePrefix(addr)
		if err != nil {
			panic(err)
		}

		if prefix.Addr().Is4() {
			out = append(out, addr)
		}
	}

	return out
}

// ipv6Prefixes returns the subset of addrs that are IPv6 prefixes.
func ipv6Prefixes(addrs []string) []string {
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		prefix, err := netip.ParsePrefix(addr)
		if err != nil {
			panic(err)
		}

		if prefix.Addr().Is6() {
			out = append(out, addr)
		}
	}

	return out
}

// GetIPv4Addr returns the first IPv4 address from Addrs, or an empty string if none.
func (c *ClientConfig) GetIPv4Addr() string {
	addrs := ipv4Prefixes(c.Addrs)
	if len(addrs) == 0 {
		return ""
	}

	return addrs[0]
}

// GetIPv6Addr returns the first IPv6 address from Addrs, or an empty string if none.
func (c *ClientConfig) GetIPv6Addr() string {
	addrs := ipv6Prefixes(c.Addrs)
	if len(addrs) == 0 {
		return ""
	}

	return addrs[0]
}

// GetRouteIPv4Addrs returns the IPv4 route addresses (RouteAddrs).
func (c *ClientConfig) GetRouteIPv4Addrs() []string { return ipv4Prefixes(c.RouteAddrs) }

// GetRouteIPv6Addrs returns the IPv6 route addresses (RouteAddrs).
func (c *ClientConfig) GetRouteIPv6Addrs() []string { return ipv6Prefixes(c.RouteAddrs) }

// GetExcludeIPv4Addrs returns the IPv4 exclude addresses (ExcludeAddrs).
func (c *ClientConfig) GetExcludeIPv4Addrs() []string { return ipv4Prefixes(c.ExcludeAddrs) }

// GetExcludeIPv6Addrs returns the IPv6 exclude addresses (ExcludeAddrs).
func (c *ClientConfig) GetExcludeIPv6Addrs() []string { return ipv6Prefixes(c.ExcludeAddrs) }

// GetRouteExcludeIPv4Addrs returns the IPv4 route excludes plus the server
// address, so the client's connection to the server bypasses the tunnel.
func (c *ClientConfig) GetRouteExcludeIPv4Addrs() []string {
	addrs := c.GetExcludeIPv4Addrs()

	for _, ip := range c.resolvedServerIPs {
		if ip.To4() != nil {
			addrs = append(addrs, ip.String()+"/32")
		}
	}

	return addrs
}

// GetRouteExcludeIPv6Addrs returns the IPv6 route excludes plus the server
// address, so the client's connection to the server bypasses the tunnel.
func (c *ClientConfig) GetRouteExcludeIPv6Addrs() []string {
	addrs := c.GetExcludeIPv6Addrs()

	for _, ip := range c.resolvedServerIPs {
		if ip.To4() == nil && ip.To16() != nil {
			addrs = append(addrs, ip.String()+"/128")
		}
	}

	return addrs
}

func isValidTLSPin(s string) bool {
	s = strings.ReplaceAll(s, ":", "")
	if len(s) != hex.EncodedLen(sha256.Size) {
		return false
	}

	_, err := hex.DecodeString(s)

	return err == nil
}

// Validate validates the ClientConfig fields.
func (c *ClientConfig) Validate() error {
	// Ensure ServerAddr is not empty.
	if c.ServerAddr == "" {
		return errors.New("server_addr is empty")
	}

	if utils.HasJSONUnsafeChars(c.ServerAddr) {
		return fmt.Errorf("invalid server_addr %q", c.ServerAddr)
	}

	host, port, err := net.SplitHostPort(c.ServerAddr)
	if err != nil {
		return fmt.Errorf("parsing server_addr %q: %w", c.ServerAddr, err)
	}

	if host == "" {
		return errors.New("server_addr host is empty")
	}

	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return fmt.Errorf("parsing server_addr port %q: %w", port, err)
	}

	// Ensure Auth is not empty.
	if c.Auth == "" {
		return errors.New("auth is empty")
	}

	if utils.HasJSONUnsafeChars(c.Auth) {
		return errors.New("auth contains unsafe characters")
	}

	// Ensure TLSPin is not empty.
	if c.TLSPin == "" {
		return errors.New("tls_pin is empty")
	}

	if !isValidTLSPin(c.TLSPin) {
		return fmt.Errorf("invalid tls_pin %q", c.TLSPin)
	}

	// Ensure TUNIface is not empty.
	if c.TUNIface == "" {
		return errors.New("tun_iface is empty")
	}

	// Validate Addrs (at least one address must be provided).
	if len(c.Addrs) == 0 {
		return errors.New("addrs are empty")
	}

	// Validate that each address in Addrs is a valid network prefix in CIDR notation.
	for _, addr := range c.Addrs {
		if _, err := netip.ParsePrefix(addr); err != nil {
			return fmt.Errorf("parsing addr prefix %q: %w", addr, err)
		}
	}

	// Validate RouteAddrs (each must be a valid CIDR range).
	for _, addr := range c.RouteAddrs {
		if _, err := netip.ParsePrefix(addr); err != nil {
			return fmt.Errorf("parsing route addr prefix %q: %w", addr, err)
		}
	}

	// Validate ExcludeAddrs (each must be a valid CIDR range).
	for _, addr := range c.ExcludeAddrs {
		if _, err := netip.ParsePrefix(addr); err != nil {
			return fmt.Errorf("parsing excluded addr prefix %q: %w", addr, err)
		}
	}

	// Validate DNSAddrs (must be valid IP addresses).
	for _, addr := range c.DNSAddrs {
		if net.ParseIP(addr) == nil {
			return fmt.Errorf("invalid DNS addr: parsing DNS addr %q", addr)
		}
	}

	if utils.HasJSONUnsafeChars(c.ObfsPassword) {
		return errors.New("obfs_password contains unsafe characters")
	}

	// Validate MTU (must be a non-zero value).
	if c.MTU == 0 {
		return errors.New("MTU is zero")
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
	fs.StringArrayVar(&c.Addrs, prefix+"addrs", c.Addrs, "ip addresses assigned to the hysteria2 client tun interface")
	fs.StringArrayVar(&c.RouteAddrs, prefix+"route-addrs", c.RouteAddrs, "ip ranges to route through the hysteria2 tunnel")
	fs.StringArrayVar(&c.ExcludeAddrs, prefix+"exclude-addrs", c.ExcludeAddrs, "exclude ip addresses/subnets from the hysteria2 tunnel")
	fs.StringArrayVar(&c.DNSAddrs, prefix+"dns-addrs", c.DNSAddrs, "dns servers to use while connected to the vpn")
	fs.Uint16Var(&c.MTU, prefix+"mtu", c.MTU, "maximum transmission unit size for the hysteria2 tun interface")

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

// serverIPs resolves the server address to its IP(s), using the host directly
// if it is already an IP literal.
func (c *ClientConfig) serverIPs(ctx context.Context) ([]net.IP, error) {
	host, _, err := net.SplitHostPort(c.ServerAddr)
	if err != nil {
		return nil, fmt.Errorf("splitting server addr %q: %w", c.ServerAddr, err)
	}

	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolving server host %q: %w", host, err)
	}

	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}

	return ips, nil
}

func (c *ClientConfig) resolveServerIPs(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	ips, err := c.serverIPs(ctx)
	if err != nil {
		return fmt.Errorf("resolving server ips: %w", err)
	}

	c.resolvedServerIPs = ips

	return nil
}

// DefaultClientConfig creates a default ClientConfig with predefined values.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerAddr:   "",
		Auth:         uuid.NewString(),
		TLSPin:       "",
		TUNIface:     "hyst0",
		Addrs:        []string{"100.100.100.101/30", "2001::ffff:ffff:ffff:fff1/126"},
		RouteAddrs:   []string{"0.0.0.0/0", "::/0"},
		ExcludeAddrs: []string{"127.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12", "10.0.0.0/8", "::1/128", "fe80::/10", "fd00::/8"},
		DNSAddrs:     []string{"208.67.222.222", "208.67.220.220", "2620:119:35::35", "2620:119:53::53"},
		MTU:          1420,
		ObfsPassword: "",
	}
}
