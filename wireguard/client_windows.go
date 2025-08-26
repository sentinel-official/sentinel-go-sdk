package wireguard

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// execFile returns the name of the executable file.
func (c *Client) execFile(name string) string {
	return ".\\" + filepath.Join("WireGuard", name+".exe")
}

// deviceName returns the name of the WireGuard interface.
func (c *Client) deviceName() (string, error) {
	return c.name, nil
}

// Down uninstalls the WireGuard tunnel service.
func (c *Client) Down() error {
	device, err := c.deviceName()
	if err != nil {
		return fmt.Errorf("failed to get device name: %w", err)
	}

	// Executes the command to uninstall the WireGuard tunnel service.
	cmd := exec.CommandContext(
		context.Background(),
		c.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", device))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}

// Up installs the WireGuard tunnel service.
func (c *Client) Up() error {
	// Executes the command to install the WireGuard tunnel service.
	cfgFile := c.serviceConfigFilePath()
	cmd := exec.CommandContext(
		c.ctx,
		c.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", cfgFile))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}
