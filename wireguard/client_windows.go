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
	return ".\\" + filepath.Join("WireGuard", c.name+".exe")
}

// interfaceName returns the name of the WireGuard interface.
func (c *Client) interfaceName() (string, error) {
	return c.name, nil
}

// Down uninstalls the WireGuard tunnel service.
func (c *Client) Down(ctx context.Context) error {
	iface, err := c.interfaceName()
	if err != nil {
		return fmt.Errorf("failed to get interface name: %w", err)
	}

	// Executes the command to uninstall the WireGuard tunnel service.
	cmd := exec.CommandContext(
		ctx,
		c.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", iface))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}

// Up installs the WireGuard tunnel service.
func (c *Client) Up(ctx context.Context) error {
	// Executes the command to install the WireGuard tunnel service.
	cmd := exec.CommandContext(
		ctx,
		c.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", c.configFilePath()))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}
