package openvpn

import (
	"context"
	"path/filepath"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return ".\\" + filepath.Join("OpenVPN", name+".exe")
}

// applyFirewall is a no-op on Windows; routing is handled by the openvpn config.
func (c *Client) applyFirewall(_ context.Context) error {
	return nil
}

// removeFirewall is a no-op on Windows; routing is handled by the openvpn config.
func (c *Client) removeFirewall(_ context.Context) error {
	return nil
}

// applyDNS is a no-op on Windows; DNS management is unix-only.
func (c *Client) applyDNS(_ context.Context) error {
	return nil
}

// removeDNS is a no-op on Windows; DNS management is unix-only.
func (c *Client) removeDNS(_ context.Context) error {
	return nil
}
