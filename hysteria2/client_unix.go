//go:build darwin || linux

package hysteria2

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// execFile returns the executable name.
func (c *Client) execFile(name string) string {
	return name
}

// applyFirewall installs the kill-switch iptables rules for the client.
func (c *Client) applyFirewall(ctx context.Context) error {
	rules, err := c.cfg.PostUp()
	if err != nil {
		return fmt.Errorf("building firewall rules: %w", err)
	}

	for _, rule := range rules {
		cmd := exec.CommandContext(ctx, rule[0], rule[1:]...)
		cmd.Stderr = os.Stderr
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
	rules, err := c.cfg.PreDown()
	if err != nil {
		return fmt.Errorf("building firewall rules: %w", err)
	}

	for _, rule := range rules {
		cmd := exec.CommandContext(ctx, rule[0], rule[1:]...)
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}

	return nil
}
