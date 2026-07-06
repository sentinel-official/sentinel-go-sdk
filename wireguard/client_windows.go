package wireguard

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return ".\\" + filepath.Join("WireGuard", name+".exe")
}

// startCmd returns the command to bring up the WireGuard interface.
func (c *Client) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the command to install the WireGuard tunnel service.
	cfgFile := c.serviceConfigFile()
	cmd := exec.CommandContext(
		ctx,
		c.execFile("wireguard"),
		"/installtunnelservice", cfgFile,
	)
	cmd.Stderr = os.Stderr

	return cmd, nil
}

// stopCmd returns the command to bring down the WireGuard interface.
func (c *Client) stopCmd() (*exec.Cmd, error) {
	// Build the command to uninstall the WireGuard tunnel service.
	cmd := exec.CommandContext(
		context.Background(),
		c.execFile("wireguard"),
		"/uninstalltunnelservice", c.device,
	)
	cmd.Stderr = os.Stderr

	return cmd, nil
}
