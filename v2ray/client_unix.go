//go:build darwin || linux

package v2ray

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}
