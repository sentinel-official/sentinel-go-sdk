//go:build darwin || linux

package xray

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}
