package openvpn

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"strconv"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// ServerConfig defines the configuration required to set up an OpenVPN server.
type ServerConfig struct {
	viper *viper.Viper `mapstructure:"-"`

	IPv4Addr     string `mapstructure:"ipv4_addr"`     // IPv4 address in CIDR format (e.g., 10.8.0.1/24)
	IPv6Addr     string `mapstructure:"ipv6_addr"`     // IPv6 address in CIDR format (optional)
	OutInterface string `mapstructure:"out_interface"` // OutInterface specifies the outbound interface.
	PKIDir       string `mapstructure:"-"`             // Path to the PKI directory used for certificates
	Port         string `mapstructure:"port"`          // Server port (e.g., "1194")
	Protocol     string `mapstructure:"protocol"`      // Transport protocol (either "tcp" or "udp")
	StatusFile   string `mapstructure:"-"`             // Path to OpenVPN status file
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

	return fmt.Sprintf("%s %s", network, netmask)
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

	if v == nil {
		panic(errors.New("nil port"))
	}

	return v.OutFrom
}

// Validate checks the correctness of all configuration fields.
func (c *ServerConfig) Validate() error {
	// At least one IP address (IPv4 or IPv6) must be provided.
	if c.IPv4Addr == "" && c.IPv6Addr == "" {
		return errors.New("ipv4_addr and ipv6_addr are empty")
	}

	// Validate IPv4 address if provided
	if c.IPv4Addr != "" {
		ip, ipNet, err := net.ParseCIDR(c.IPv4Addr)
		if err != nil {
			return fmt.Errorf("parsing ipv4_addr %q: %w", c.IPv4Addr, err)
		}

		if ip == nil || ipNet == nil {
			return errors.New("invalid ipv4_addr: ip or netmask is nil")
		}
	}

	// Validate IPv6 address if provided
	if c.IPv6Addr != "" {
		ip, ipNet, err := net.ParseCIDR(c.IPv6Addr)
		if err != nil {
			return fmt.Errorf("parsing ipv6_addr %q: %w", c.IPv6Addr, err)
		}

		if ip == nil || ipNet == nil {
			return errors.New("invalid ipv6_addr: ip or netmask is nil")
		}
	}

	// Ensure OutInterface is not empty.
	if c.OutInterface == "" {
		return errors.New("out_interface is empty")
	}

	// PKI directory is mandatory
	if c.PKIDir == "" {
		return errors.New("pki_dir is empty")
	}

	// Port must be valid and non-empty
	if c.Port == "" {
		return errors.New("port is empty")
	}

	if _, err := netip.NewPortFromString(c.Port); err != nil {
		return fmt.Errorf("parsing port %q: %w", c.Port, err)
	}

	// Protocol must be either "tcp" or "udp"
	validProtocols := map[string]bool{
		"tcp": true,
		"udp": true,
	}
	if !validProtocols[c.Protocol] {
		return fmt.Errorf("unsupported protocol %q (allowed: tcp, udp)", c.Protocol)
	}

	// StatusFile must be non-empty
	if c.StatusFile == "" {
		return errors.New("status_file is empty")
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

// SetForFlags is a placeholder method to allow binding ServerConfig to CLI flags.
func (c *ServerConfig) SetForFlags(_ *pflag.FlagSet, _ string) {}

// DefaultServerConfig returns a ServerConfig instance populated with randomly generated values.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		IPv4Addr:     fmt.Sprintf("10.%d.%d.1/24", rand.IntN(256), rand.IntN(256)),
		IPv6Addr:     "",
		OutInterface: "eth0",
		PKIDir:       "",
		Port:         strconv.FormatUint(uint64(utils.RandomPort()), 10),
		Protocol:     randomProtocol(),
		StatusFile:   "",
	}
}

// randomProtocol randomly returns either "tcp" or "udp".
func randomProtocol() string {
	return [...]string{
		"tcp", "udp",
	}[rand.IntN(2)]
}
