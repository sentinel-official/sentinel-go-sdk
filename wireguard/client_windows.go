package wireguard

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return ".\\" + filepath.Join("WireGuard", name+".exe")
}

// startCmd returns the command to bring up the WireGuard interface.
func (c *Client) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the command to uninstall the WireGuard tunnel service.
	cmd := exec.CommandContext(
		ctx,
		c.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", c.device))...,
	)

	return cmd, nil
}

// stopCmd returns the command to bring down the WireGuard interface.
func (c *Client) stopCmd() (*exec.Cmd, error) {
	// Build the command to install the WireGuard tunnel service.
	cfgFile := c.serviceConfigFile()
	cmd := exec.Command(
		c.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", cfgFile))...,
	)

	return cmd, nil
}
