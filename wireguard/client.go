package wireguard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Client implements the types.ClientService interface.
var _ types.ClientService = (*Client)(nil)

// Client represents a WireGuard client with associated home directory and name.
type Client struct {
	homeDir string // Home directory for client files.
	name    string // Name of the interface.
}

// NewClient creates a new Client instance.
func NewClient(appDir string) *Client {
	return &Client{
		homeDir: filepath.Join(appDir, "wireguard"),
		name:    "wg0",
	}
}

// WithName sets the name for the client and returns the updated Client instance.
func (c *Client) WithName(name string) *Client {
	c.name = name
	return c
}

// appConfigFilePath returns the full path to the application's configuration file.
func (c *Client) appConfigFilePath() string {
	return filepath.Join(c.homeDir, "config.toml")
}

// serviceConfigFilePath returns the full path to the service-specific configuration file.
func (c *Client) serviceConfigFilePath() string {
	return filepath.Join(c.homeDir, fmt.Sprintf("%s.conf", c.name))
}

// Type returns the service type of the client.
func (c *Client) Type() types.ServiceType {
	return types.ServiceTypeWireGuard
}

// Init sets up service configuration, creating directories and writing defaults unless config exists.
func (c *Client) Init(force bool) error {
	// Create the home directory if it doesn't exist
	if err := os.MkdirAll(c.homeDir, 0755); err != nil {
		return fmt.Errorf("failed to create home directory: %w", err)
	}

	// Construct the full path to the config file
	cfgFile := c.appConfigFilePath()

	// Check if the config file exists at the specified path
	cfgFileExists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}

	// Write default config only if file doesn't exist or force flag is enabled
	if !cfgFileExists || force {
		cfg := DefaultClientConfig()
		if err := cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	return nil
}

// IsUp checks if the WireGuard interface is up.
func (c *Client) IsUp(ctx context.Context) (bool, error) {
	// Retrieves the interface name.
	iface, err := c.interfaceName()
	if err != nil {
		return false, fmt.Errorf("failed to get interface name: %w", err)
	}

	// Executes the 'wg show' command to check the interface status.
	cmd := exec.CommandContext(
		ctx,
		c.execFile("wg"),
		strings.Fields(fmt.Sprintf("show %s", iface))...,
	)

	// Capture stderr output.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command and handle errors.
	if err := cmd.Run(); err != nil {
		// Check if the error matches "No such device".
		if strings.Contains(stderr.String(), "No such device") {
			return false, nil
		}

		return false, fmt.Errorf("failed to run command: %w", err)
	}

	return true, nil
}

// PreUp writes the configuration to the config file before starting the client process.
func (c *Client) PreUp() error {
	// Initialize viper instance
	v := viper.New()

	// Construct the full path to the config file
	cfgFile := c.appConfigFilePath()

	// Check if the config file exists at the specified path
	cfgFileExists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}

	// If the config file exists, proceed to read its contents
	if cfgFileExists {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal configuration into the config object
	cfg := DefaultClientConfig()
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to validate config file: %w", err)
	}

	// Writes configuration to file.
	cfgFile = c.serviceConfigFilePath()
	if err := cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// PostUp performs operations after the client process is started.
func (c *Client) PostUp() error {
	return nil
}

// PreDown performs operations before the client process is terminated.
func (c *Client) PreDown() error {
	return nil
}

// PostDown performs cleanup operations after the client process is terminated.
func (c *Client) PostDown() error {
	// Removes configuration file.
	cfgFile := c.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("failed to remove config: %w", err)
	}

	return nil
}

// Statistics returns the download and upload statistics for the WireGuard interface.
func (c *Client) Statistics(ctx context.Context) (int64, int64, error) {
	// Retrieves the interface name.
	iface, err := c.interfaceName()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get interface name: %w", err)
	}

	// Executes the 'wg show' command to get transfer statistics.
	output, err := exec.CommandContext(
		ctx,
		c.execFile("wg"),
		strings.Fields(fmt.Sprintf("show %s transfer", iface))...,
	).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to run command: %w", err)
	}

	// Split the command output into lines and process each line.
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		columns := strings.Split(line, "\t")
		if len(columns) != 3 {
			continue
		}

		// Parse upload traffic stats.
		uploadBytes, err := strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse upload bytes: %w", err)
		}

		// Parse download traffic stats.
		downloadBytes, err := strconv.ParseInt(columns[2], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse download bytes: %w", err)
		}

		return uploadBytes, downloadBytes, nil
	}

	return 0, 0, nil // Return 0 statistics if no data found.
}
