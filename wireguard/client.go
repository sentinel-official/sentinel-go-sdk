package wireguard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
func NewClient() *Client {
	return &Client{}
}

// WithHomeDir sets the home directory for the client and returns the updated Client instance.
func (c *Client) WithHomeDir(homeDir string) *Client {
	c.homeDir = homeDir
	return c
}

// WithName sets the name for the client and returns the updated Client instance.
func (c *Client) WithName(name string) *Client {
	c.name = name
	return c
}

// configFilePath returns the file path of the client's configuration file.
func (c *Client) configFilePath() string {
	return filepath.Join(c.homeDir, fmt.Sprintf("%s.conf", c.name))
}

// Type returns the service type of the client.
func (c *Client) Type() types.ServiceType {
	return types.ServiceTypeWireGuard
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
func (c *Client) PreUp(v interface{}) error {
	// Checks for valid parameter type.
	cfg, ok := v.(*ClientConfig)
	if !ok {
		return fmt.Errorf("invalid parameter type %T", v)
	}

	// Writes configuration to file.
	if err := cfg.WriteToFile(c.configFilePath()); err != nil {
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
	if err := utils.RemoveFile(c.configFilePath()); err != nil {
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
