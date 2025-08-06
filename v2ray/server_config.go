package v2ray

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// InboundServerConfig represents the V2Ray inbound server configuration options.
type InboundServerConfig struct {
	Port        string `mapstructure:"port"`          // Port defines the inbound port range.
	Proxy       string `mapstructure:"proxy"`         // Proxy defines the protocol used (e.g., vmess).
	Security    string `mapstructure:"security"`      // Security specifies the encryption method.
	TLSCertPath string `mapstructure:"tls_cert_path"` // TLSCertPath specifies the path to the TLS certificate.
	TLSKeyPath  string `mapstructure:"tls_key_path"`  // TLSKeyPath specifies the path to the TLS private key.
	Transport   string `mapstructure:"transport"`     // Transport specifies the transport protocol.
}

// GetPort parses and returns the port configuration.
func (c *InboundServerConfig) GetPort() *netip.Port {
	port, err := netip.NewPortFromString(c.Port)
	if err != nil {
		panic(err)
	}

	return port
}

// InPort returns the inbound port range.
func (c *InboundServerConfig) InPort() string {
	return c.GetPort().InPort()
}

// OutPort returns the outbound port range.
func (c *InboundServerConfig) OutPort() string {
	return c.GetPort().OutPort()
}

// Tag creates a Tag instance based on the InboundServerConfig configuration.
func (c *InboundServerConfig) Tag() *Tag {
	proxy := NewProxyProtocolFromString(c.Proxy)
	security := NewTransportSecurityFromString(c.Security)
	transport := NewTransportProtocolFromString(c.Transport)

	return &Tag{
		Port:      c.GetPort(),
		Proxy:     proxy,
		Security:  security,
		Transport: transport,
	}
}

// Validate validates the InboundServerConfig fields.
func (c *InboundServerConfig) Validate() error {
	// Ensure Port is not empty.
	if c.Port == "" {
		return errors.New("port cannot be empty")
	}

	// Validate the Port value.
	if _, err := netip.NewPortFromString(c.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	// Validate the Proxy protocol.
	if v := NewProxyProtocolFromString(c.Proxy); !v.IsValid() {
		return fmt.Errorf("invalid proxy %s", v)
	}

	// Validate the Security setting.
	security := NewTransportSecurityFromString(c.Security)
	if !security.IsValid() {
		return fmt.Errorf("invalid security %s", security)
	}

	// Ensure TLS paths are provided if Security is TLS.
	if security == TransportSecurityTLS {
		if c.TLSCertPath == "" {
			return errors.New("tls_cert_path cannot be empty")
		}
		if c.TLSKeyPath == "" {
			return errors.New("tls_key_path cannot be empty")
		}
	}

	// Validate the Transport protocol.
	if v := NewTransportProtocolFromString(c.Transport); !v.IsValid() {
		return fmt.Errorf("invalid transport %s", v)
	}

	return nil
}

// ServerConfig represents the V2Ray server configuration options.
type ServerConfig struct {
	Inbounds []*InboundServerConfig `mapstructure:"inbounds"` // Inbounds is a list of inbound server configurations.
}

// Validate validates the ServerConfig fields.
func (c *ServerConfig) Validate() error {
	// Ensure Inbounds is not empty.
	if len(c.Inbounds) == 0 {
		return errors.New("inbounds cannot be empty")
	}

	// Create sets to track unique inbound and outbound ports and tags.
	inPortSet := make(map[uint16]bool)
	outPortSet := make(map[uint16]bool)
	tagSet := make(map[string]bool)

	// Validate each InboundServerConfig.
	for _, inbound := range c.Inbounds {
		if err := inbound.Validate(); err != nil {
			return fmt.Errorf("invalid inbound: %w", err)
		}

		// Parse the port range and check for duplicates.
		port, err := netip.NewPortFromString(inbound.Port)
		if err != nil {
			panic(err)
		}
		if port == nil {
			panic(errors.New("nil port"))
		}

		// Check inbound ports for duplicates.
		for p := port.InFrom; p <= port.InTo; p++ {
			if inPortSet[p] {
				return fmt.Errorf("duplicate in port %d", p)
			}
			inPortSet[p] = true
		}

		// Check outbound ports for duplicates.
		for p := port.OutFrom; p <= port.OutTo; p++ {
			if outPortSet[p] {
				return fmt.Errorf("duplicate out port %d", p)
			}
			outPortSet[p] = true
		}

		// Check tags for duplicates.
		tag := inbound.Tag().String()
		if tagSet[tag] {
			return fmt.Errorf("duplicate tag %s", tag)
		}
		tagSet[tag] = true
	}

	return nil
}

// WriteServiceConfig generates the service-level configuration file using the service template.
func (c *ServerConfig) WriteServiceConfig(filename string) error {
	// Load the service template from the embedded or filesystem path.
	text, err := fs.ReadFile("server.json.tmpl")
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

// SetForFlags adds server configuration flags to the specified FlagSet.
func (c *ServerConfig) SetForFlags(_ *pflag.FlagSet) {}

// DefaultServerConfig creates a default ServerConfig with predefined values.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Inbounds: []*InboundServerConfig{
			{
				Port:        fmt.Sprintf("%d", utils.RandomPort()),
				Proxy:       "vmess",
				Security:    "none",
				TLSCertPath: "",
				TLSKeyPath:  "",
				Transport:   "grpc",
			},
			{
				Port:        fmt.Sprintf("%d", utils.RandomPort()),
				Proxy:       "vmess",
				Security:    "none",
				TLSCertPath: "",
				TLSKeyPath:  "",
				Transport:   "tcp",
			},
		},
	}
}
