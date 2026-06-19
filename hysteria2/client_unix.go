//go:build darwin || linux

package hysteria2

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}
