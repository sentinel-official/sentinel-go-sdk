package hysteria2

import (
	"context"
	"errors"
	"path/filepath"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return ".\\" + filepath.Join("Hysteria2", name+".exe")
}

// ifaceStats returns the TUN interface rx/tx byte counters.
// Windows implementation is not yet supported.
func (c *Client) ifaceStats(_ context.Context, _ string) (int64, int64, error) {
	return 0, 0, errors.New("interface statistics not implemented on Windows")
}
