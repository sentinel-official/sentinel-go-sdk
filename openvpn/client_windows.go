package openvpn

import (
	"path/filepath"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return ".\\" + filepath.Join("OpenVPN", name+".exe")
}
