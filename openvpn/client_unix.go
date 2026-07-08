//go:build darwin || linux

package openvpn

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}

// applyFirewall installs the kill-switch iptables rules for the client.
func (c *Client) applyFirewall(ctx context.Context) error {
	rules, err := c.cfg.PostUp(ctx)
	if err != nil {
		return fmt.Errorf("building firewall rules: %w", err)
	}

	for _, rule := range rules {
		cmd := exec.CommandContext(ctx, rule[0], rule[1:]...)

		if err := cmd.Run(); err != nil {
			// Roll back any rules already applied so a partial apply does not
			// leave a dangling kill-switch that blocks host traffic.
			_ = c.removeFirewall(ctx)

			return fmt.Errorf("running firewall rule %v: %w", rule, err)
		}
	}

	return nil
}

// removeFirewall removes the kill-switch iptables rules for the client.
// Teardown is best-effort: individual delete failures are ignored.
func (c *Client) removeFirewall(ctx context.Context) error {
	rules, err := c.cfg.PreDown(ctx)
	if err != nil {
		return fmt.Errorf("building firewall rules: %w", err)
	}

	for _, rule := range rules {
		cmd := exec.CommandContext(ctx, rule[0], rule[1:]...)
		_ = cmd.Run()
	}

	return nil
}

// waitForInterface waits until the named network interface exists or ctx is done.
func waitForInterface(ctx context.Context, name string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if _, err := net.InterfaceByName(name); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for interface %q: %w", name, ctx.Err())
		case <-ticker.C:
		}
	}
}

// applyDNS sets the system DNS to the configured servers via resolvconf.
func (c *Client) applyDNS(ctx context.Context) error {
	if len(c.cfg.DNSAddrs) == 0 {
		return nil
	}

	// Wait for the TUN interface to exist before configuring DNS on it.
	if err := waitForInterface(ctx, c.cfg.TUNIface); err != nil {
		return fmt.Errorf("waiting for interface %q: %w", c.cfg.TUNIface, err)
	}

	var stdin bytes.Buffer
	for _, addr := range c.cfg.DNSAddrs {
		fmt.Fprintf(&stdin, "nameserver %s\n", addr)
	}

	cmd := exec.CommandContext(ctx, "resolvconf", "-a", c.cfg.TUNIface, "-m", "0", "-x")
	cmd.Stdin = &stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running resolvconf: %w", err)
	}

	return nil
}

// removeDNS reverts the resolvconf DNS entry for the interface (best-effort).
func (c *Client) removeDNS(ctx context.Context) error {
	if len(c.cfg.DNSAddrs) == 0 {
		return nil
	}

	cmd := exec.CommandContext(ctx, "resolvconf", "-d", c.cfg.TUNIface, "-f")
	_ = cmd.Run()

	return nil
}
