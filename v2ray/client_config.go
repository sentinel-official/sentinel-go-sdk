package v2ray

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/v2fly/v2ray-core/v5/common/uuid"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

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
func (c *ClientConfig) SetForFlags(f *pflag.FlagSet) {
	f.StringVar(&c.Name, "v2ray.name", c.Name, "name of the v2ray client instance")
	f.Uint16Var(&c.API.Port, "v2ray.api.port", c.API.Port, "port for the v2ray statistics and management operations")
	f.Uint16Var(&c.Proxy.Port, "v2ray.proxy.port", c.Proxy.Port, "port for the v2ray socks5 proxy server")
}

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
