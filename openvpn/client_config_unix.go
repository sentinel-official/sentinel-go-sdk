//go:build darwin || linux

package openvpn

import (
	"context"
	"fmt"
	"net"
)

// firewallRules builds the kill-switch rules: ACCEPT exceptions first
// (acceptAction), then the DROP last (dropAction) so exceptions apply first.
func (c *ClientConfig) firewallRules(acceptAction, dropAction string, ips []net.IP) [][]string {
	addrs := c.GetExcludeAddrs()
	rules := make([][]string, 0, 4+len(addrs)+len(ips))

	// Allow loopback traffic.
	rules = append(rules,
		[]string{"iptables", acceptAction, "OUTPUT", "-o", "lo", "-j", "ACCEPT"},  // ACCEPT loopback for IPv4.
		[]string{"ip6tables", acceptAction, "OUTPUT", "-o", "lo", "-j", "ACCEPT"}, // ACCEPT loopback for IPv6.
	)

	// Allow each excluded address.
	for _, addr := range addrs {
		execName := "iptables"
		if addr.Addr().Is6() {
			execName = "ip6tables"
		}

		rules = append(rules, []string{execName, acceptAction, "OUTPUT", "!", "-o", c.TUNIface, "-d", addr.String(), "-j", "ACCEPT"})
	}

	// Allow the resolved server address(es), so the client can reach the server.
	for _, ip := range ips {
		execName := "iptables"
		if ip.To4() == nil {
			execName = "ip6tables"
		}

		rules = append(rules, []string{execName, acceptAction, "OUTPUT", "!", "-o", c.TUNIface, "-d", ip.String(), "-j", "ACCEPT"})
	}

	// Drop any output that is not leaving the TUN interface. Emitted last so the
	// ACCEPT exceptions above are in place before the DROP takes effect.
	rules = append(rules,
		[]string{"iptables", dropAction, "OUTPUT", "!", "-o", c.TUNIface, "-j", "DROP"},  // DROP rule for IPv4.
		[]string{"ip6tables", dropAction, "OUTPUT", "!", "-o", c.TUNIface, "-j", "DROP"}, // DROP rule for IPv6.
	)

	return rules
}

// PostUp generates the iptables kill-switch rules to install after the client starts.
func (c *ClientConfig) PostUp(ctx context.Context) ([][]string, error) {
	ips, err := c.serverIPs(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving server ips: %w", err)
	}

	return c.firewallRules("-I", "-A", ips), nil
}

// PreDown generates the iptables rules to remove the kill-switch. Server-IP
// resolution is best-effort so teardown always proceeds.
func (c *ClientConfig) PreDown(ctx context.Context) ([][]string, error) {
	ips, _ := c.serverIPs(ctx)

	return c.firewallRules("-D", "-D", ips), nil
}

// serverIPs resolves the server address to its IP addresses.
func (c *ClientConfig) serverIPs(ctx context.Context) ([]net.IP, error) {
	if ip := net.ParseIP(c.Addr); ip != nil {
		return []net.IP{ip}, nil
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, c.Addr)
	if err != nil {
		return nil, fmt.Errorf("resolving server host %q: %w", c.Addr, err)
	}

	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ips = append(ips, addr.IP)
	}

	return ips, nil
}
