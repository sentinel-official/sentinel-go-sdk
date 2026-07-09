//go:build darwin || linux

package wireguard

import (
	"fmt"
	"net/netip"
)

func (c *ClientConfig) hasDefaultRoute() bool {
	for _, addr := range c.Peer.AllowAddrs {
		prefix, err := netip.ParsePrefix(addr)
		if err != nil {
			continue
		}

		if prefix.Bits() == 0 {
			return true
		}
	}

	return false
}

// PostUp generates PostUp rules for IPv4 and IPv6 settings.
func (c *ClientConfig) PostUp() []string {
	if !c.hasDefaultRoute() {
		return nil
	}

	// Get the list of excluded IP addresses.
	addrs := c.GetExcludeAddrs()
	matchRule := fmt.Sprintf("! -o %s -m mark ! --mark $(wg show %s fwmark)", c.Name, c.Name)
	rules := make([]string, 0, 2+len(addrs))
	rules = append(rules,
		fmt.Sprintf("iptables -I OUTPUT %s -j DROP", matchRule),  // Add DROP rule for IPv4.
		fmt.Sprintf("ip6tables -I OUTPUT %s -j DROP", matchRule), // Add DROP rule for IPv6.
	)

	// Add ACCEPT rules for each excluded address.
	for _, v := range addrs {
		execName := "iptables"
		if v.Addr().Is6() {
			execName = "ip6tables"
		}

		rules = append(rules, fmt.Sprintf("%s -I OUTPUT %s -d %s -j ACCEPT", execName, matchRule, v))
	}

	return rules
}

// PreDown generates PreDown rules to remove the PostUp rules for IPv4 and IPv6.
func (c *ClientConfig) PreDown() []string {
	if !c.hasDefaultRoute() {
		return nil
	}

	// Get the list of excluded IP addresses.
	addrs := c.GetExcludeAddrs()
	matchRule := fmt.Sprintf("! -o %s -m mark ! --mark $(wg show %s fwmark)", c.Name, c.Name)
	rules := make([]string, 0, 2+len(addrs))
	rules = append(rules,
		fmt.Sprintf("iptables -D OUTPUT %s -j DROP", matchRule),  // Delete DROP rule for IPv4.
		fmt.Sprintf("ip6tables -D OUTPUT %s -j DROP", matchRule), // Delete DROP rule for IPv6.
	)

	// Delete ACCEPT rules for each excluded address.
	for _, v := range addrs {
		execName := "iptables"
		if v.Addr().Is6() {
			execName = "ip6tables"
		}

		rules = append(rules, fmt.Sprintf("%s -D OUTPUT %s -d %s -j ACCEPT", execName, matchRule, v))
	}

	return rules
}
