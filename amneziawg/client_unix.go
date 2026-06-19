//go:build darwin || linux

package amneziawg

import (
	"context"
	"os/exec"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}

// startCmd returns the command to bring up the AmneziaWG interface.
func (c *Client) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the 'awg-quick up' command with the service config file.
	cfgFile := c.serviceConfigFile()
	cmd := exec.CommandContext(
		ctx,
		c.execFile("awg-quick"),
		"up", cfgFile,
	)

	return cmd, nil
}

// stopCmd returns the command to bring down the AmneziaWG interface.
func (c *Client) stopCmd() (*exec.Cmd, error) {
	// Build the 'awg-quick down' command with the service config file.
	cfgFile := c.serviceConfigFile()
	cmd := exec.CommandContext(
		context.Background(),
		c.execFile("awg-quick"),
		"down", cfgFile,
	)

	return cmd, nil
}
