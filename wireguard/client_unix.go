//go:build darwin || linux

package wireguard

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}

// startCmd returns the command to bring up the WireGuard interface.
func (c *Client) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the 'wg-quick up' command with the service config file.
	cfgFile := c.serviceConfigFile()
	cmd := exec.CommandContext(
		ctx,
		c.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("up %s", cfgFile))...,
	)

	return cmd, nil
}

// stopCmd returns the command to bring down the WireGuard interface.
func (c *Client) stopCmd() (*exec.Cmd, error) {
	// Build the 'wg-quick down' command with the service config file.
	cfgFile := c.serviceConfigFile()
	cmd := exec.Command(
		c.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("down %s", cfgFile))...,
	)

	return cmd, nil
}
