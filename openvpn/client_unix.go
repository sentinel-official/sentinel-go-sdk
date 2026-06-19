//go:build darwin || linux

package openvpn

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}
