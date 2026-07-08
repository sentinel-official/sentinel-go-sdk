package v2ray

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

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
		return errors.New("port is zero")
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
	Addr              string `mapstructure:"-"` // Addr specifies the destination server address.
	Port              uint16 `mapstructure:"-"` // Port specifies the destination server port.
	ProxyProtocol     string `mapstructure:"-"` // ProxyProtocol specifies the proxy protocol to use.
	TransportProtocol string `mapstructure:"-"` // TransportProtocol specifies the transport protocol to use.
	TransportSecurity string `mapstructure:"-"` // TransportSecurity specifies the transport security type.
	TLSPin            string `mapstructure:"-"` // TLSPin is the SHA-256 pin of the server's TLS certificate.
}

// Validate validates the OutboundClientConfig fields.
func (c *OutboundClientConfig) Validate() error {
	// Ensure the address is not empty.
	if c.Addr == "" {
		return errors.New("addr is empty")
	}

	// Ensure Port is not empty.
	if c.Port == 0 {
		return errors.New("port is zero")
	}

	// Validate the Proxy protocol.
	if v := NewProxyProtocolFromString(c.ProxyProtocol); !v.IsValid() {
		return fmt.Errorf("invalid proxy_protocol %q", v)
	}

	// Validate the Transport protocol.
	if v := NewTransportProtocolFromString(c.TransportProtocol); !v.IsValid() {
		return fmt.Errorf("invalid transport_protocol %q", v)
	}

	// Validate the Transport Security.
	if v := NewTransportSecurityFromString(c.TransportSecurity); !v.IsValid() {
		return fmt.Errorf("invalid transport_security %q", v)
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
		return errors.New("port is zero")
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
	viper *viper.Viper `mapstructure:"-"`

	API       *APIClientConfig        `mapstructure:"api"`   // API defines the API client configuration.
	ID        string                  `mapstructure:"-"`     // ID specifies the client identifier in UUID format.
	Outbounds []*OutboundClientConfig `mapstructure:"-"`     // Outbounds defines the list of outbound connection configurations.
	Proxy     *ProxyClientConfig      `mapstructure:"proxy"` // Proxy defines the proxy client configuration.
}

// GetID parses and returns the UUID from the ClientConfig's ID field.
// It panics if the ID is not a valid UUID string.
func (c *ClientConfig) GetID() uuid.UUID {
	id, err := uuid.Parse(c.ID)
	if err != nil {
		panic(err)
	}

	return id
}

// Validate validates the ClientConfig fields.
func (c *ClientConfig) Validate() error {
	// Validate the API client configuration.
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("validating API config: %w", err)
	}

	// Ensure the ID is not empty.
	if c.ID == "" {
		return errors.New("id is empty")
	}

	// Validate each outbound client configuration.
	for _, outbound := range c.Outbounds {
		if err := outbound.Validate(); err != nil {
			return fmt.Errorf("validating outbound config: %w", err)
		}
	}

	// Validate the proxy client configuration.
	if err := c.Proxy.Validate(); err != nil {
		return fmt.Errorf("validating proxy config: %w", err)
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

	// Unmarshal the config data into the ServerConfig struct.
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
	text, err := fs.ReadFile("client.json.tmpl")
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

	fs.Uint16Var(&c.API.Port, prefix+"api.port", c.API.Port, "port for the v2ray statistics and management operations")
	fs.Uint16Var(&c.Proxy.Port, prefix+"proxy.port", c.Proxy.Port, "port for the v2ray socks5 proxy server")

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

// DefaultClientConfig creates a default ClientConfig with predefined values.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		API:       DefaultAPIClientConfig(),
		ID:        NewStringUUID(),
		Outbounds: []*OutboundClientConfig{},
		Proxy:     DefaultProxyClientConfig(),
	}
}
