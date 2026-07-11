//go:build darwin || linux

package hysteria2

import (
	"net"
)

// firewallRules builds the kill-switch rules: ACCEPT exceptions first
// (acceptAction), then the DROP last (dropAction) so exceptions apply first.
func (c *ClientConfig) firewallRules(acceptAction, dropAction string, ips []net.IP) [][]string {
	rules := make([][]string, 0, 4+len(c.GetExcludeIPv4Addrs())+len(c.GetExcludeIPv6Addrs())+len(ips))

	// Allow loopback traffic.
	rules = append(rules,
		[]string{"iptables", acceptAction, "OUTPUT", "-o", "lo", "-j", "ACCEPT"},  // ACCEPT loopback for IPv4.
		[]string{"ip6tables", acceptAction, "OUTPUT", "-o", "lo", "-j", "ACCEPT"}, // ACCEPT loopback for IPv6.
	)

	// Allow each excluded address.
	for _, addr := range c.GetExcludeIPv4Addrs() {
		rules = append(rules, []string{"iptables", acceptAction, "OUTPUT", "!", "-o", c.TUNIface, "-d", addr, "-j", "ACCEPT"})
	}

	for _, addr := range c.GetExcludeIPv6Addrs() {
		rules = append(rules, []string{"ip6tables", acceptAction, "OUTPUT", "!", "-o", c.TUNIface, "-d", addr, "-j", "ACCEPT"})
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
func (c *ClientConfig) PostUp() [][]string {
	return c.firewallRules("-I", "-A", c.resolvedServerIPs)
}

// PreDown generates the iptables rules to remove the kill-switch.
func (c *ClientConfig) PreDown() [][]string {
	return c.firewallRules("-D", "-D", c.resolvedServerIPs)
}
