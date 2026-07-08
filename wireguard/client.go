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

	"github.com/sentinel-official/sentinel-go-sdk/process"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Client implements the types.ClientService interface.
var _ types.ClientService = (*Client)(nil)

// Client represents the WireGuard client service instance.
type Client struct {
	*process.Manager // Embedded process manager for handling lifecycle.

	cfg     *ClientConfig // Configuration settings for the service.
	device  string        // Name of the network interface.
	homeDir string        // Home directory of the service.
}

// NewClient creates a new Client instance.
func NewClient(name, appDir string, cfg *ClientConfig) *Client {
	return &Client{
		Manager: process.NewManager(name),
		cfg:     cfg,
		device:  defaultDevice,
		homeDir: filepath.Join(appDir, "wireguard"),
	}
}

// WithDevice sets the WireGuard network interface name and returns the updated Client instance.
func (c *Client) WithDevice(device string) *Client {
	c.device = device

	return c
}

// Type returns the service type of the client.
func (c *Client) Type() types.ServiceType {
	return types.ServiceTypeWireGuard
}

// IsRunning checks if the WireGuard interface is up and active.
func (c *Client) IsRunning() (bool, error) {
	// Executes the 'wg show' command to check the interface status.
	cmd := exec.CommandContext(
		context.Background(),
		c.execFile("wg"),
		"show", c.device,
	)

	// Capture stderr output.
	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	// Run the command and handle errors.
	if err := cmd.Run(); err != nil {
		// Treat a missing interface or absent kernel module (userspace fallback) as not running.
		if out := stderr.String(); strings.Contains(out, "No such device") || strings.Contains(out, "Protocol not supported") {
			return false, nil
		}

		return false, fmt.Errorf("running command: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return true, nil
}

// Init sets up client configuration, creating directories and writing defaults unless config exists.
func (c *Client) Init(force bool) error {
	// Create the home directory if it doesn't exist.
	if err := os.MkdirAll(c.homeDir, 0700); err != nil {
		return fmt.Errorf("creating home directory %q: %w", c.homeDir, err)
	}

	// Construct the full path to the config file.
	cfgFile := c.appConfigFile()

	// Check if the config file exists at the specified path.
	exists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
	}

	// Write config only if file doesn't exist or force flag is enabled.
	if !exists || force {
		if err := c.cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("writing app config file %q: %w", cfgFile, err)
		}
	}

	return nil
}

// Setup prepares the WireGuard client service for operation.
func (c *Client) Setup(ctx context.Context) error {
	return c.Manager.Setup(ctx, func() error { //nolint:wrapcheck
		// Construct the full path to the config file.
		cfgFile := c.appConfigFile()

		// Check if the config file exists at the specified path.
		exists, err := utils.IsFileExists(cfgFile)
		if err != nil {
			return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
		}

		// If the config file exists, proceed to read its contents.
		if exists {
			if err := c.cfg.ReadAppConfig(cfgFile); err != nil {
				return fmt.Errorf("reading app config file %q: %w", cfgFile, err)
			}
		}

		// Validate the config.
		if err := c.cfg.Validate(); err != nil {
			return fmt.Errorf("validating config: %w", err)
		}

		// Write configuration to file.
		cfgFile = c.serviceConfigFile()
		if err := c.cfg.WriteServiceConfig(cfgFile); err != nil {
			return fmt.Errorf("writing service config file %q: %w", cfgFile, err)
		}

		return nil
	})
}

// Start starts the WireGuard client service.
func (c *Client) Start(parent context.Context) (context.Context, error) {
	return c.Manager.Start(parent, func(ctx context.Context) error { //nolint:wrapcheck
		// Start the WireGuard process.
		cmd, err := c.startCmd(ctx)
		if err != nil {
			return fmt.Errorf("preparing command: %w", err)
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running command: %w", err)
		}

		return nil
	})
}

// Stop stops the WireGuard client service.
func (c *Client) Stop() error {
	return c.Manager.Stop(func() error { //nolint:wrapcheck
		cmd, err := c.stopCmd()
		if err != nil {
			return fmt.Errorf("preparing command: %w", err)
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running command: %w", err)
		}

		return nil
	})
}

// Wait waits for all background goroutines to complete.
func (c *Client) Wait(ctx context.Context) error {
	return c.Manager.Wait(ctx, nil) //nolint:wrapcheck
}

// Cleanup removes client-specific configuration files.
func (c *Client) Cleanup() error {
	return c.Manager.Cleanup(func() error { //nolint:wrapcheck
		// Removes configuration file.
		cfgFile := c.serviceConfigFile()
		if err := utils.RemoveFile(cfgFile); err != nil {
			return fmt.Errorf("removing service config file %q: %w", cfgFile, err)
		}

		return nil
	})
}

// Statistics retrieves the latest transfer statistics from the WireGuard interface.
func (c *Client) Statistics(ctx context.Context) (int64, int64, error) {
	// Executes the 'wg show' command to get transfer statistics.
	cmd := exec.CommandContext(
		ctx,
		c.execFile("wg"),
		"show", c.device, "transfer",
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("running command: %w", err)
	}

	// Split the command output into lines and process each line.
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		columns := strings.Split(line, "\t")
		if len(columns) != 3 {
			continue
		}

		// Parse peer upload traffic stats.
		rxBytes, err := strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parsing rx bytes %q: %w", columns[1], err)
		}

		// Parse peer download traffic stats.
		txBytes, err := strconv.ParseInt(columns[2], 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parsing tx bytes %q: %w", columns[2], err)
		}

		return rxBytes, txBytes, nil
	}

	return 0, 0, nil
}

func (c *Client) appConfigFile() string     { return filepath.Join(c.homeDir, "config.toml") }
func (c *Client) serviceConfigFile() string { return filepath.Join(c.homeDir, c.device+".conf") }
