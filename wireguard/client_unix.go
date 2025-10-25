//go:build darwin || linux

package wireguard

import (
	"context"
	"os/exec"
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
		"up", cfgFile,
	)

	return cmd, nil
}

// stopCmd returns the command to bring down the WireGuard interface.
func (c *Client) stopCmd() (*exec.Cmd, error) {
	// Build the 'wg-quick down' command with the service config file.
	cfgFile := c.serviceConfigFile()
	cmd := exec.CommandContext(
		context.Background(),
		c.execFile("wg-quick"),
		"down", cfgFile,
	)

	return cmd, nil
}
