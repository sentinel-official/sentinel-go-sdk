package v2ray

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/v2fly/v2ray-core/v5/common/uuid"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// APIClientConfig represents the configuration for the API client.
type APIClientConfig struct {
	Port uint16 `mapstructure:"port"` // Port specifies the port for the API client.
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
		Port: 2323,
	}
}

// OutboundClientConfig represents the configuration for outbound connections.
type OutboundClientConfig struct {
	Addr              string `mapstructure:"addr"`               // Addr specifies the destination server address.
	Port              uint16 `mapstructure:"port"`               // Port specifies the destination server port.
	ProxyProtocol     string `mapstructure:"proxy_protocol"`     // ProxyProtocol specifies the proxy protocol to use.
	TransportProtocol string `mapstructure:"transport_protocol"` // TransportProtocol specifies the transport protocol to use.
	TransportSecurity string `mapstructure:"transport_security"` // TransportSecurity specifies the transport security type.
}

// Validate validates the OutboundClientConfig fields.
func (c *OutboundClientConfig) Validate() error {
	// Ensure the address is not empty.
	if c.Addr == "" {
		return errors.New("addr cannot be empty")
	}

	// Ensure Port is not empty.
	if c.Port == 0 {
		return errors.New("port cannot be empty")
	}

	// Validate the Proxy protocol.
	if v := NewProxyProtocolFromString(c.ProxyProtocol); !v.IsValid() {
		return fmt.Errorf("invalid proxy %s", v)
	}

	// Validate the Transport protocol.
	if v := NewTransportProtocolFromString(c.TransportProtocol); !v.IsValid() {
		return fmt.Errorf("invalid transport %s", v)
	}

	// Validate the Transport Security.
	if v := NewTransportSecurityFromString(c.TransportSecurity); !v.IsValid() {
		return fmt.Errorf("invalid security %s", v)
	}

	return nil
}

// GetProxyProtocol returns the proxy protocol as a ProxyProtocol type.
func (c *OutboundClientConfig) GetProxyProtocol() ProxyProtocol {
	return NewProxyProtocolFromString(c.ProxyProtocol)
}

// GetTransportProtocol returns the transport protocol as a TransportProtocol type.
func (c *OutboundClientConfig) GetTransportProtocol() TransportProtocol {
	return NewTransportProtocolFromString(c.TransportProtocol)
}

// GetTransportSecurity returns the transport security as a TransportSecurity type.
func (c *OutboundClientConfig) GetTransportSecurity() TransportSecurity {
	return NewTransportSecurityFromString(c.TransportSecurity)
}

// GetPort returns the parsed port configuration.
func (c *OutboundClientConfig) GetPort() *netip.Port {
	return &netip.Port{
		InFrom:  c.Port,
		InTo:    c.Port,
		OutFrom: c.Port,
		OutTo:   c.Port,
	}
}

// Tag generates a unique tag string based on the outbound connection configuration.
func (c *OutboundClientConfig) Tag() string {
	items := []string{
		c.Addr,
		strconv.Itoa(int(c.Port)),
		c.GetProxyProtocol().String(),
		c.GetTransportProtocol().String(),
		c.GetTransportSecurity().String(),
	}

	return strings.Join(items, "_")
}

// ProxyClientConfig represents the proxy client configuration.
type ProxyClientConfig struct {
	Port uint16 `mapstructure:"port"` // Port specifies the port for the proxy client.
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
		Port: 1080,
	}
}

// ClientConfig represents the V2Ray client configuration options.
type ClientConfig struct {
	API       *APIClientConfig        `mapstructure:"api"`       // API defines the API client configuration.
	ID        string                  `mapstructure:"id"`        // ID specifies the client identifier in UUID format.
	Outbounds []*OutboundClientConfig `mapstructure:"outbounds"` // Outbounds defines the list of outbound connection configurations.
	Proxy     *ProxyClientConfig      `mapstructure:"proxy"`     // Proxy defines the proxy client configuration.
}

// GetID parses and returns the UUID from the ClientConfig's ID field.
// It panics if the ID is not a valid UUID string.
func (c *ClientConfig) GetID() uuid.UUID {
	id, err := uuid.ParseString(c.ID)
	if err != nil {
		panic(err)
	}

	return id
}

// Validate validates the ClientConfig fields.
func (c *ClientConfig) Validate() error {
	// Validate the API client configuration.
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("invalid api config: %w", err)
	}

	// Ensure the ID is not empty.
	if c.ID == "" {
		return errors.New("id cannot be empty")
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

// WriteServiceConfig generates the service-level configuration file using the service template.
func (c *ClientConfig) WriteServiceConfig(filename string) error {
	// Load the service template from the embedded or filesystem path.
	text, err := fs.ReadFile("client.json.tmpl")
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
	f.Uint16Var(&c.API.Port, "v2ray.api.port", c.API.Port, "port for the v2ray statistics and management operations")
	f.Uint16Var(&c.Proxy.Port, "v2ray.proxy.port", c.Proxy.Port, "port for the v2ray socks5 proxy server")
}

// DefaultClientConfig creates a default ClientConfig with predefined values.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		API:       DefaultAPIClientConfig(),
		ID:        NewStringUUID(),
		Outbounds: []*OutboundClientConfig{},
		Proxy:     DefaultProxyClientConfig(),
	}
}
