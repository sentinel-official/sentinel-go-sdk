package openvpn

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"

	"github.com/spf13/pflag"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// ServerConfig defines the configuration required to set up an OpenVPN server.
type ServerConfig struct {
	IPv4Addr   string `mapstructure:"ipv4_addr"` // IPv4 address in CIDR format (e.g., 10.8.0.1/24)
	IPv6Addr   string `mapstructure:"ipv6_addr"` // IPv6 address in CIDR format (optional)
	PKIDir     string `mapstructure:"-"`         // Path to the PKI directory used for certificates
	Port       string `mapstructure:"port"`      // Server port (e.g., "1194")
	Protocol   string `mapstructure:"protocol"`  // Transport protocol (either "tcp" or "udp")
	StatusFile string `mapstructure:"-"`         // Path to OpenVPN status file
}

// ExtIPv4Addr returns the IPv4 address and netmask extracted from the CIDR.
func (c *ServerConfig) ExtIPv4Addr() string {
	ip, ipNet, err := net.ParseCIDR(c.IPv4Addr)
	if err != nil || ip == nil || ipNet == nil {
		return ""
	}

	// Extract netmask and network from CIDR notation
	netmask := net.IP(ipNet.Mask)
	network := ip.Mask(ipNet.Mask)

	return fmt.Sprintf("%s %s\n", network, netmask)
}

// ExtIPv6Addr returns the configured IPv6 address.
func (c *ServerConfig) ExtIPv6Addr() string {
	return c.IPv6Addr
}

// OutPort returns the outbound port as a uint16 value.
func (c *ServerConfig) OutPort() uint16 {
	v, err := netip.NewPortFromString(c.Port)
	if err != nil {
		panic(err)
	}

	return v.OutFrom
}

// Validate checks the correctness of all configuration fields.
func (c *ServerConfig) Validate() error {
	// At least one IP address (IPv4 or IPv6) must be provided.
	if c.IPv4Addr == "" && c.IPv6Addr == "" {
		return errors.New("either ipv4_addr or ipv6_addr is required")
	}

	// Validate IPv4 address if provided
	if c.IPv4Addr != "" {
		ip, ipNet, err := net.ParseCIDR(c.IPv4Addr)
		if err != nil {
			return fmt.Errorf("invalid ipv4_addr: %w", err)
		}
		if ip == nil || ipNet == nil {
			return errors.New("invalid ipv4_addr: either ip or netmask is empty")
		}
	}

	// Validate IPv6 address if provided
	if c.IPv6Addr != "" {
		ip, ipNet, err := net.ParseCIDR(c.IPv6Addr)
		if err != nil {
			return fmt.Errorf("invalid ipv6_addr: %w", err)
		}
		if ip == nil || ipNet == nil {
			return errors.New("invalid ipv6_addr: either ip or netmask is empty")
		}
	}

	// PKI directory is mandatory
	if c.PKIDir == "" {
		return errors.New("pki_dir cannot be empty")
	}

	// Port must be valid and non-empty
	if c.Port == "" {
		return errors.New("port cannot be empty")
	}
	if _, err := netip.NewPortFromString(c.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	// Protocol must be either "tcp" or "udp"
	validProtocols := map[string]bool{
		"tcp": true,
		"udp": true,
	}
	if !validProtocols[c.Protocol] {
		return fmt.Errorf("protocol must be one of: tcp, udp")
	}

	// StatusFile must be non-empty
	if c.StatusFile == "" {
		return errors.New("status_file cannot be empty")
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

// SetForFlags is a placeholder method to allow binding ServerConfig to CLI flags.
func (c *ServerConfig) SetForFlags(_ *pflag.FlagSet) {}

// DefaultServerConfig returns a ServerConfig instance populated with randomly generated values.
func DefaultServerConfig() *ServerConfig {

	return &ServerConfig{
		IPv4Addr:   fmt.Sprintf("10.%d.%d.1/24", rand.Intn(256), rand.Intn(256)),
		IPv6Addr:   "",
		PKIDir:     "",
		Port:       fmt.Sprintf("%d", utils.RandomPort()),
		Protocol:   randomProtocol(),
		StatusFile: "",
	}
}

// randomProtocol randomly returns either "tcp" or "udp"
func randomProtocol() string {
	return [...]string{"tcp", "udp"}[rand.Intn(2)]
}
