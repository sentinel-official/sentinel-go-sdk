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

	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Client implements the types.ClientService interface.
var _ types.ClientService = (*Client)(nil)

// Client represents a WireGuard client with associated home directory and name.
type Client struct {
	cfg     *ClientConfig // Configuration settings for the WireGuard client.
	homeDir string        // Home directory for client files.
	name    string        // Name of the interface.

	cancel context.CancelFunc // Context cancel function to stop background tasks.
	ctx    context.Context    // Context for server lifecycle management.
	eg     *errgroup.Group    // Error group for managing background goroutines.
}

// NewClient creates a new Client instance.
func NewClient(appDir string, cfg *ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Client{
		cfg:     cfg,
		homeDir: filepath.Join(appDir, "wireguard"),
		name:    "wg0",
		cancel:  cancel,
		ctx:     ctx,
		eg:      eg,
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
		return fmt.Errorf("creating home directory %q: %w", c.homeDir, err)
	}

	// Construct the full path to the config file
	cfgFile := c.appConfigFilePath()

	// Check if the config file exists at the specified path
	exists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
	}

	// Write config only if file doesn't exist or force flag is enabled
	if !exists || force {
		if err := c.cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("writing app config file %q: %w", cfgFile, err)
		}
	}

	return nil
}

// IsUp checks if the WireGuard interface is up.
func (c *Client) IsUp() (bool, error) {
	// Retrieves the device name.
	device, err := c.deviceName()
	if err != nil {
		return false, fmt.Errorf("getting device name: %w", err)
	}
	if device == "" {
		return false, nil
	}

	// Executes the 'wg show' command to check the interface status.
	cmd := exec.Command(
		c.execFile("wg"),
		strings.Fields(fmt.Sprintf("show %s", device))...,
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

		return false, fmt.Errorf("running command: %w", err)
	}

	return true, nil
}

// PreUp writes the configuration to the config file before starting the client process.
func (c *Client) PreUp() error {
	// Construct the full path to the config file
	cfgFile := c.appConfigFilePath()

	// Check if the config file exists at the specified path
	exists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
	}

	// If the config file exists, proceed to read its contents
	if exists {
		if err := c.cfg.ReadAppConfig(cfgFile); err != nil {
			return fmt.Errorf("reading app config file %q: %w", cfgFile, err)
		}
	}

	// Validate the config
	if err := c.cfg.Validate(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	// Write configuration to file.
	cfgFile = c.serviceConfigFilePath()
	if err := c.cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("writing service config file %q: %w", cfgFile, err)
	}

	return nil
}

// PostUp performs operations after the client process is started.
func (c *Client) PostUp() error {
	return nil
}

// Wait waits for all background goroutines to complete.
func (c *Client) Wait() error {
	if err := c.eg.Wait(); err != nil {
		if !utils.ErrorIs(context.Cause(c.ctx), context.Canceled) {
			return err
		}
	}

	return nil
}

// PreDown performs operations before the client process is terminated.
func (c *Client) PreDown() error {
	// Cancel background tasks if any.
	c.cancel()

	return nil
}

// PostDown performs cleanup operations after the client process is terminated.
func (c *Client) PostDown() error {
	// Removes configuration file.
	cfgFile := c.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("removing service config file %q: %w", cfgFile, err)
	}

	return nil
}

// Statistics returns the download and upload statistics for the WireGuard interface.
func (c *Client) Statistics(ctx context.Context) (int64, int64, error) {
	// Retrieves the device name.
	device, err := c.deviceName()
	if err != nil {
		return 0, 0, fmt.Errorf("getting device name: %w", err)
	}
	if device == "" {
		return 0, 0, fmt.Errorf("device name is empty")
	}

	// Executes the 'wg show' command to get transfer statistics.
	output, err := exec.CommandContext(
		ctx,
		c.execFile("wg"),
		strings.Fields(fmt.Sprintf("show %s transfer", device))...,
	).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("running command: %w", err)
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
			return 0, 0, fmt.Errorf("parsing upload bytes %q: %w", columns[1], err)
		}

		// Parse download traffic stats.
		downloadBytes, err := strconv.ParseInt(columns[2], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parsing download bytes %q: %w", columns[2], err)
		}

		return uploadBytes, downloadBytes, nil
	}

	return 0, 0, nil // Return 0 statistics if no data found.
}
