package v2ray

import (
	"embed"
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/v2fly/v2ray-core/v5/common/uuid"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Embed the template files for V2Ray configurations.
//
//go:embed *.tmpl
var fs embed.FS

// APIClientConfig represents the configuration for the API client.
type APIClientConfig struct {
	Port uint16 `mapstructure:"port"`
}

// Validate validates the APIClientConfig fields.
func (c *APIClientConfig) Validate() error {
	// Ensure Port is not empty.
	if c.Port == 0 {
		return errors.New("port cannot be empty")
	}

	return nil
}

// DefaultAPIClientConfig creates a default API client configuration.
func DefaultAPIClientConfig() *APIClientConfig {
	return &APIClientConfig{
		Port: utils.RandomPort(),
	}
}

// OutboundClientConfig represents the configuration for outbound connections.
type OutboundClientConfig struct {
	Port      uint16 `mapstructure:"port"`
	Proxy     string `mapstructure:"proxy"`
	Security  string `mapstructure:"security"`
	Transport string `mapstructure:"transport"`
}

// Validate validates the OutboundClientConfig fields.
func (c *OutboundClientConfig) Validate() error {
	// Ensure Port is not empty.
	if c.Port == 0 {
		return errors.New("port cannot be empty")
	}

	// Validate the Proxy protocol.
	if v := NewProxyProtocolFromString(c.Proxy); !v.IsValid() {
		return fmt.Errorf("invalid proxy %s", v)
	}

	// Validate the Security setting.
	if v := NewTransportSecurityFromString(c.Security); !v.IsValid() {
		return fmt.Errorf("invalid security %s", v)
	}

	// Validate the Transport protocol.
	if v := NewTransportProtocolFromString(c.Transport); !v.IsValid() {
		return fmt.Errorf("invalid transport %s", v)
	}

	return nil
}

// GetPort returns the parsed port configuration.
func (c *OutboundClientConfig) GetPort() types.Port {
	return types.Port{
		InFrom:  c.Port,
		InTo:    c.Port,
		OutFrom: c.Port,
		OutTo:   c.Port,
	}
}

// Tag generates a tag based on the outbound configuration.
func (c *OutboundClientConfig) Tag() *Tag {
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

// ProxyClientConfig represents the proxy client configuration.
type ProxyClientConfig struct {
	Port uint16 `mapstructure:"port"`
}

// Validate validates the ProxyClientConfig fields.
func (c *ProxyClientConfig) Validate() error {
	// Ensure Port is not empty.
	if c.Port == 0 {
		return errors.New("port cannot be empty")
	}

	return nil
}

// DefaultProxyClientConfig creates a default ProxyClientConfig.
func DefaultProxyClientConfig() *ProxyClientConfig {
	return &ProxyClientConfig{
		Port: utils.RandomPort(),
	}
}

// ClientConfig represents the V2Ray client configuration options.
type ClientConfig struct {
	Addr      string                  `mapstructure:"addr"`
	API       *APIClientConfig        `mapstructure:"api"`
	ID        string                  `mapstructure:"id"`
	Name      string                  `mapstructure:"name"`
	Outbounds []*OutboundClientConfig `mapstructure:"outbounds"`
	Proxy     *ProxyClientConfig      `mapstructure:"proxy"`
}

func (c *ClientConfig) GetID() uuid.UUID {
	id, err := uuid.ParseString(c.ID)
	if err != nil {
		panic(err)
	}

	return id
}

// Validate validates the ClientConfig fields.
func (c *ClientConfig) Validate() error {
	// Ensure the address is not empty.
	if c.Addr == "" {
		return errors.New("addr cannot be empty")
	}

	// Validate the API client configuration.
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("invalid api config: %w", err)
	}

	// Ensure the ID is not empty.
	if c.ID == "" {
		return errors.New("id cannot be empty")
	}

	// Ensure the Name is not empty.
	if c.Name == "" {
		return errors.New("name cannot be empty")
	}

	// Validate each outbound client configuration.
	for _, outbound := range c.Outbounds {
		if err := outbound.Validate(); err != nil {
			return fmt.Errorf("invalid outbound: %w", err)
		}
	}

	// Validate the proxy client configuration.
	if err := c.Proxy.Validate(); err != nil {
		return fmt.Errorf("invalid proxy config: %w", err)
	}

	return nil
}

// WriteToFile writes the client configuration to a file.
func (c *ClientConfig) WriteToFile(name string) error {
	// Read the client configuration template file.
	text, err := fs.ReadFile("client.json.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Execute the template and write it to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, name); err != nil {
		return fmt.Errorf("failed to execute template to file: %w", err)
	}

	return nil
}

// SetForFlags adds client configuration flags to the specified FlagSet.
func (c *ClientConfig) SetForFlags(_ *pflag.FlagSet) {}

// DefaultClientConfig creates a default ClientConfig with predefined values.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Addr:      "",
		API:       DefaultAPIClientConfig(),
		ID:        NewStringUUID(),
		Name:      "v2ray",
		Outbounds: []*OutboundClientConfig{},
		Proxy:     DefaultProxyClientConfig(),
	}
}

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
func (c *InboundServerConfig) GetPort() types.Port {
	port, err := types.NewPortFromString(c.Port)
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
	if _, err := types.NewPortFromString(c.Port); err != nil {
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
		port, err := types.NewPortFromString(inbound.Port)
		if err != nil {
			panic(err)
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

// WriteToFile writes the server configuration to a file.
func (c *ServerConfig) WriteToFile(name string) error {
	// Read the server configuration template file.
	text, err := fs.ReadFile("server.json.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Execute the template and write it to the specified file.
	if err := utils.ExecTemplateToFile(string(text), c, name); err != nil {
		return fmt.Errorf("failed to execute template to file: %w", err)
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
