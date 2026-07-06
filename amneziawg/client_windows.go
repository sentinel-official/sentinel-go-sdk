package amneziawg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return ".\\" + filepath.Join("AmneziaWG", name+".exe")
}

// startCmd returns the command to bring up the AmneziaWG interface.
func (c *Client) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the command to install the AmneziaWG tunnel service.
	cfgFile := c.serviceConfigFile()
	cmd := exec.CommandContext(
		ctx,
		c.execFile("amneziawg"),
		"/installtunnelservice", cfgFile,
	)
	cmd.Stderr = os.Stderr

	return cmd, nil
}

// stopCmd returns the command to bring down the AmneziaWG interface.
func (c *Client) stopCmd() (*exec.Cmd, error) {
	// Build the command to uninstall the AmneziaWG tunnel service.
	cmd := exec.CommandContext(
		context.Background(),
		c.execFile("amneziawg"),
		"/uninstalltunnelservice", c.device,
	)
	cmd.Stderr = os.Stderr

	return cmd, nil
}
