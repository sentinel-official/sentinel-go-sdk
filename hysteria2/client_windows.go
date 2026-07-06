package hysteria2

import (
	"context"
	"path/filepath"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return ".\\" + filepath.Join("Hysteria2", name+".exe")
}

// applyFirewall is a no-op on Windows; routing is handled by the hysteria route config.
func (c *Client) applyFirewall(_ context.Context) error {
	return nil
}

// removeFirewall is a no-op on Windows; routing is handled by the hysteria route config.
func (c *Client) removeFirewall(_ context.Context) error {
	return nil
}
