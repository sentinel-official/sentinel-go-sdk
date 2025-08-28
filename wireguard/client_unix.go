//go:build darwin || linux

package wireguard

import (
	"fmt"
	"os/exec"
	"strings"
)

// execFile returns the name of the executable file.
func (c *Client) execFile(name string) string {
	return name
}

// Down shuts down the WireGuard interface.
func (c *Client) Down() error {
	// Executes the 'wg-quick down' command to bring down the interface.
	cfgFile := c.serviceConfigFilePath()
	cmd := exec.Command(
		c.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("down %s", cfgFile))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	return nil
}

// Up starts the WireGuard interface.
func (c *Client) Up() error {
	// Executes the 'wg-quick up' command to bring up the interface.
	cfgFile := c.serviceConfigFilePath()
	cmd := exec.CommandContext(
		c.ctx,
		c.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("up %s", cfgFile))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	return nil
}
