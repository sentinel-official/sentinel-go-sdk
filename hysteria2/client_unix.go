//go:build darwin || linux

package hysteria2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}

// ifaceStats reads the TUN interface rx/tx byte counters from the OS.
// On Linux, the counters are read from /sys/class/net/<iface>/statistics/{rx_bytes,tx_bytes}.
// Host RX maps to download, host TX maps to upload.
func (c *Client) ifaceStats(_ context.Context, iface string) (int64, int64, error) {
	// Read rx_bytes (download).
	rxPath := filepath.Clean(filepath.Join("/sys/class/net", iface, "statistics/rx_bytes"))

	rxData, err := os.ReadFile(rxPath)
	if err != nil {
		return 0, 0, fmt.Errorf("reading rx_bytes for interface %q: %w", iface, err)
	}

	rxBytes, err := strconv.ParseInt(strings.TrimSpace(string(rxData)), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing rx_bytes for interface %q: %w", iface, err)
	}

	// Read tx_bytes (upload).
	txPath := filepath.Clean(filepath.Join("/sys/class/net", iface, "statistics/tx_bytes"))

	txData, err := os.ReadFile(txPath)
	if err != nil {
		return 0, 0, fmt.Errorf("reading tx_bytes for interface %q: %w", iface, err)
	}

	txBytes, err := strconv.ParseInt(strings.TrimSpace(string(txData)), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing tx_bytes for interface %q: %w", iface, err)
	}

	return rxBytes, txBytes, nil
}
