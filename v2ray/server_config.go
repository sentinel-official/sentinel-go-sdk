package v2ray

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// InboundServerConfig represents the V2Ray inbound server configuration options.
type InboundServerConfig struct {
	Port              string `mapstructure:"port"`               // Port defines the inbound port range.
	ProxyProtocol     string `mapstructure:"proxy_protocol"`     // ProxyProtocol defines the protocol used (e.g., vmess).
	TransportProtocol string `mapstructure:"transport_protocol"` // TransportProtocol specifies the transport protocol.
	TransportSecurity string `mapstructure:"transport_security"` // TransportSecurity specifies the encryption method.
}

// GetPort parses and returns the port configuration.
func (c *InboundServerConfig) GetPort() *netip.Port {
	port, err := netip.NewPortFromString(c.Port)
	if err != nil {
		panic(err)
	}

	if port == nil {
		panic(errors.New("nil port"))
	}

	return port
}

// GetProxyProtocol parses and returns the proxy protocol configuration.
func (c *InboundServerConfig) GetProxyProtocol() ProxyProtocol {
	return NewProxyProtocolFromString(c.ProxyProtocol)
}

// GetTransportProtocol parses and returns the transport protocol configuration.
func (c *InboundServerConfig) GetTransportProtocol() TransportProtocol {
	return NewTransportProtocolFromString(c.TransportProtocol)
}

// GetTransportSecurity parses and returns the transport security configuration.
func (c *InboundServerConfig) GetTransportSecurity() TransportSecurity {
	return NewTransportSecurityFromString(c.TransportSecurity)
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
func (c *InboundServerConfig) Tag() string {
	return c.InPort()
}

// Validate validates the InboundServerConfig fields.
func (c *InboundServerConfig) Validate() error {
	// Ensure Port is not empty.
	if c.Port == "" {
		return errors.New("port is empty")
	}

	// Validate the Port value.
	if _, err := netip.NewPortFromString(c.Port); err != nil {
		return fmt.Errorf("parsing port %q: %w", c.Port, err)
	}

	// Validate the Proxy protocol.
	if v := NewProxyProtocolFromString(c.ProxyProtocol); !v.IsValid() {
		return fmt.Errorf("invalid proxy_protocol %q", v)
	}

	// Validate the Transport protocol.
	if v := NewTransportProtocolFromString(c.TransportProtocol); !v.IsValid() {
		return fmt.Errorf("invalid transport_protocol %q", v)
	}

	// Validate the Transport security.
	if v := NewTransportSecurityFromString(c.TransportSecurity); !v.IsValid() {
		return fmt.Errorf("invalid transport_security %q", v)
	}

	if c.GetTransportProtocol() == TransportProtocolQUIC && c.GetTransportSecurity() != TransportSecurityTLS {
		return errors.New("quic requires tls")
	}

	return nil
}

// ServerConfig represents the V2Ray server configuration options.
type ServerConfig struct {
	viper *viper.Viper `mapstructure:"-"`

	Inbounds    []*InboundServerConfig `mapstructure:"inbounds"` // Inbounds is a list of inbound server configurations.
	TLSCertFile string                 `mapstructure:"-"`        // TLSCertFile is the path to the TLS certificate file.
	TLSKeyFile  string                 `mapstructure:"-"`        // TLSKeyFile is the path to the TLS private key file.
}

// Validate validates the ServerConfig fields.
func (c *ServerConfig) Validate() error {
	// Ensure Inbounds is not empty.
	if len(c.Inbounds) == 0 {
		return errors.New("inbounds are empty")
	}

	// Create sets to track unique inbound and outbound ports and tags.
	inPortSet := make(map[uint16]bool)
	outPortSet := make(map[uint16]bool)
	tagSet := make(map[string]bool)

	// Validate each InboundServerConfig.
	for _, inbound := range c.Inbounds {
		if err := inbound.Validate(); err != nil {
			return fmt.Errorf("validating inbound: %w", err)
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
		for p := int(port.InFrom); p <= int(port.InTo); p++ {
			if inPortSet[uint16(p)] {
				return fmt.Errorf("duplicate in_port %d", p)
			}

			inPortSet[uint16(p)] = true
		}

		// Check outbound ports for duplicates.
		for p := int(port.OutFrom); p <= int(port.OutTo); p++ {
			if outPortSet[uint16(p)] {
				return fmt.Errorf("duplicate out_port %d", p)
			}

			outPortSet[uint16(p)] = true
		}

		// Check tags for duplicates.
		tag := inbound.Tag()
		if tagSet[tag] {
			return fmt.Errorf("duplicate tag %q", tag)
		}

		tagSet[tag] = true
	}

	// Validate TLS certificate file is specified.
	if c.TLSCertFile == "" {
		return errors.New("tls_cert_file is empty")
	}

	// Validate TLS key file is specified.
	if c.TLSKeyFile == "" {
		return errors.New("tls_key_file is empty")
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
	text, err := fs.ReadFile("server.json.tmpl")
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

// SetForFlags adds server configuration flags to the specified FlagSet.
func (c *ServerConfig) SetForFlags(_ *pflag.FlagSet, _ string) {}

// DefaultServerConfig creates a default ServerConfig with predefined values.
func DefaultServerConfig() *ServerConfig {
	// Start with the two strongest inbounds (VLESS over TLS on WebSocket and on
	// gRPC), then add three random ones.
	inbounds := make([]*InboundServerConfig, 0, 5)
	inbounds = append(inbounds,
		&InboundServerConfig{
			Port:              strconv.FormatUint(uint64(utils.RandomPort()), 10),
			ProxyProtocol:     ProxyProtocolVLess.String(),
			TransportProtocol: TransportProtocolWebSocket.String(),
			TransportSecurity: TransportSecurityTLS.String(),
		},
		&InboundServerConfig{
			Port:              strconv.FormatUint(uint64(utils.RandomPort()), 10),
			ProxyProtocol:     ProxyProtocolVLess.String(),
			TransportProtocol: TransportProtocolGRPC.String(),
			TransportSecurity: TransportSecurityTLS.String(),
		},
	)

	for range 3 {
		inbounds = append(inbounds, randomInboundServerConfig())
	}

	return &ServerConfig{
		Inbounds: inbounds,
	}
}

// randomInboundServerConfig builds an inbound with a random proxy protocol and
// transport, always secured with TLS.
func randomInboundServerConfig() *InboundServerConfig {
	return &InboundServerConfig{
		Port:              strconv.FormatUint(uint64(utils.RandomPort()), 10),
		ProxyProtocol:     randomProxyProtocol().String(),
		TransportProtocol: randomTransportProtocol().String(),
		TransportSecurity: TransportSecurityTLS.String(),
	}
}

// randomProxyProtocol returns a random proxy protocol type (vless or vmess).
func randomProxyProtocol() ProxyProtocol {
	return [...]ProxyProtocol{
		ProxyProtocolVLess,
		ProxyProtocolVMess,
	}[rand.IntN(2)]
}

// randomTransportProtocol returns a random transport protocol from available options.
func randomTransportProtocol() TransportProtocol {
	return [...]TransportProtocol{
		TransportProtocolGUN,
		TransportProtocolGRPC,
		TransportProtocolHTTP,
		TransportProtocolMKCP,
		TransportProtocolQUIC,
		TransportProtocolTCP,
		TransportProtocolWebSocket,
	}[rand.IntN(7)]
}
