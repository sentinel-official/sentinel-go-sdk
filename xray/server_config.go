package xray

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

// InboundServerConfig represents the Xray inbound server configuration options.
type InboundServerConfig struct {
	Port              string   `mapstructure:"port"`               // Port defines the inbound port range.
	ProxyProtocol     string   `mapstructure:"proxy_protocol"`     // ProxyProtocol defines the protocol used (e.g., vless).
	TransportProtocol string   `mapstructure:"transport_protocol"` // TransportProtocol specifies the transport protocol.
	TransportSecurity string   `mapstructure:"transport_security"` // TransportSecurity specifies the encryption method.
	Flow              string   `mapstructure:"flow"`               // Flow specifies the VLESS flow control setting.
	Method            string   `mapstructure:"method"`             // Method specifies the Shadowsocks 2022 method.
	Key               string   `mapstructure:"-"`                  // Key is the generated Shadowsocks 2022 server-level key (iPSK).
	SeedKey           string   `mapstructure:"-"`                  // SeedKey is the throwaway Shadowsocks 2022 seed-user key.
	Reality           *Reality `mapstructure:"reality"`            // Reality specifies the Reality security configuration.
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

// GetFlow parses and returns the flow configuration.
func (c *InboundServerConfig) GetFlow() Flow {
	return NewFlowFromString(c.Flow)
}

// GetMethod returns the Shadowsocks 2022 method, defaulting when empty.
func (c *InboundServerConfig) GetMethod() string {
	if c.Method == "" {
		return ShadowsocksMethod
	}

	return c.Method
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

	// Validate the Flow when set.
	if c.Flow != "" {
		if v := NewFlowFromString(c.Flow); !v.IsValid() {
			return fmt.Errorf("invalid flow %q", v)
		}
	}

	// Validate the Method for Shadowsocks 2022 inbounds.
	if c.GetProxyProtocol() == ProxyProtocolShadowsocks2022 {
		if c.Method != "" && c.Method != ShadowsocksMethod {
			return fmt.Errorf("invalid method %q, only %q is supported", c.Method, ShadowsocksMethod)
		}
	}

	// Validate the Reality configuration when used.
	if c.GetTransportSecurity() == TransportSecurityReality {
		if c.Reality == nil {
			return errors.New("reality is nil")
		}

		if err := c.Reality.Validate(); err != nil {
			return fmt.Errorf("validating reality: %w", err)
		}
	}

	return nil
}

// ServerConfig represents the Xray server configuration options.
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
		for p := port.InFrom; p <= port.InTo; p++ {
			if inPortSet[p] {
				return fmt.Errorf("duplicate in_port %d", p)
			}

			inPortSet[p] = true
		}

		// Check outbound ports for duplicates.
		for p := port.OutFrom; p <= port.OutTo; p++ {
			if outPortSet[p] {
				return fmt.Errorf("duplicate out_port %d", p)
			}

			outPortSet[p] = true
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
	// Start with the two strongest inbounds (VLESS+Reality with Vision over raw
	// TCP, and VLESS+Reality over XHTTP), then add three random ones.
	inbounds := make([]*InboundServerConfig, 0, 5)
	inbounds = append(inbounds,
		&InboundServerConfig{
			Port:              strconv.FormatUint(uint64(utils.RandomPort()), 10),
			ProxyProtocol:     ProxyProtocolVLess.String(),
			TransportProtocol: TransportProtocolTCP.String(),
			TransportSecurity: TransportSecurityReality.String(),
			Flow:              FlowVision.String(),
		},
		&InboundServerConfig{
			Port:              strconv.FormatUint(uint64(utils.RandomPort()), 10),
			ProxyProtocol:     ProxyProtocolVLess.String(),
			TransportProtocol: TransportProtocolXHTTP.String(),
			TransportSecurity: TransportSecurityReality.String(),
		},
	)

	for range 3 {
		inbounds = append(inbounds, randomInboundServerConfig())
	}

	return &ServerConfig{
		Inbounds: inbounds,
	}
}

// randomInboundServerConfig builds an inbound with a random proxy protocol,
// transport, and security, enabling Vision flow only for vless over raw TCP.
func randomInboundServerConfig() *InboundServerConfig {
	proxyProtocol := randomProxyProtocol()

	// Shadowsocks 2022 encrypts at the proxy layer, so it runs over raw TCP with
	// no stream-level security.
	if proxyProtocol == ProxyProtocolShadowsocks2022 {
		return &InboundServerConfig{
			Port:              strconv.FormatUint(uint64(utils.RandomPort()), 10),
			ProxyProtocol:     proxyProtocol.String(),
			TransportProtocol: TransportProtocolTCP.String(),
			TransportSecurity: TransportSecurityNone.String(),
		}
	}

	transportProtocol := randomTransportProtocol()
	transportSecurity := randomTransportSecurity(transportProtocol)

	flow := ""
	if proxyProtocol == ProxyProtocolVLess &&
		transportProtocol == TransportProtocolTCP &&
		(transportSecurity == TransportSecurityTLS || transportSecurity == TransportSecurityReality) {
		flow = FlowVision.String()
	}

	return &InboundServerConfig{
		Port:              strconv.FormatUint(uint64(utils.RandomPort()), 10),
		ProxyProtocol:     proxyProtocol.String(),
		TransportProtocol: transportProtocol.String(),
		TransportSecurity: transportSecurity.String(),
		Flow:              flow,
	}
}

// randomProxyProtocol returns a random proxy protocol type.
func randomProxyProtocol() ProxyProtocol {
	return [...]ProxyProtocol{
		ProxyProtocolVLess,
		ProxyProtocolVMess,
		ProxyProtocolTrojan,
		ProxyProtocolShadowsocks2022,
	}[rand.IntN(4)]
}

// randomTransportProtocol returns a random transport protocol from available options.
func randomTransportProtocol() TransportProtocol {
	return [...]TransportProtocol{
		TransportProtocolTCP,
		TransportProtocolWebSocket,
		TransportProtocolGRPC,
		TransportProtocolHTTPUpgrade,
		TransportProtocolXHTTP,
	}[rand.IntN(5)]
}

// randomTransportSecurity returns tls, or reality when the transport supports it.
func randomTransportSecurity(transport TransportProtocol) TransportSecurity {
	options := []TransportSecurity{TransportSecurityTLS}
	if transport == TransportProtocolTCP ||
		transport == TransportProtocolGRPC ||
		transport == TransportProtocolXHTTP {
		options = append(options, TransportSecurityReality)
	}

	return options[rand.IntN(len(options))]
}
