//go:build darwin || linux

package wireguard

import (
	"fmt"
	"strings"
)

// PostUp generates PostUp rules for IPv4 and IPv6 settings.
func (c *ClientConfig) PostUp() string {
	// Get the list of excluded IP addresses.
	addrs := c.GetExcludeAddrs()
	matchRule := fmt.Sprintf("! -o %s -m mark ! --mark $(wg show %s fwmark)", c.Name, c.Name)
	rules := []string{
		fmt.Sprintf("iptables -I OUTPUT %s -j DROP", matchRule),  // Add DROP rule for IPv4.
		fmt.Sprintf("ip6tables -I OUTPUT %s -j DROP", matchRule), // Add DROP rule for IPv6.
	}

	// Add ACCEPT rules for each excluded address.
	for _, v := range addrs {
		execName := "iptables"
		if v.Addr().Is6() {
			execName = "ip6tables"
		}

		rules = append(rules, fmt.Sprintf("%s -I OUTPUT %s -d %s -j ACCEPT", execName, matchRule, v))
	}

	return strings.Join(rules, "; ")
}

// PreDown generates PreDown rules to remove the PostUp rules for IPv4 and IPv6.
func (c *ClientConfig) PreDown() string {
	// Get the list of excluded IP addresses.
	addrs := c.GetExcludeAddrs()
	matchRule := fmt.Sprintf("! -o %s -m mark ! --mark $(wg show %s fwmark)", c.Name, c.Name)
	rules := []string{
		fmt.Sprintf("iptables -D OUTPUT %s -j DROP", matchRule),  // Delete DROP rule for IPv4.
		fmt.Sprintf("ip6tables -D OUTPUT %s -j DROP", matchRule), // Delete DROP rule for IPv6.
	}

	// Delete ACCEPT rules for each excluded address.
	for _, v := range addrs {
		execName := "iptables"
		if v.Addr().Is6() {
			execName = "ip6tables"
		}

		rules = append(rules, fmt.Sprintf("%s -D OUTPUT %s -d %s -j ACCEPT", execName, matchRule, v))
	}

	return strings.Join(rules, "; ")
}

// PostDown generates the PostDown rules based on IPv4 and IPv6 settings
func (c *ServerConfig) PostDown() string {
	// Initialize an empty slice to store the rules
	var rules []string

	// Check if an IPv4 address is configured
	if c.IPv4Addr != "" {
		rules = append(rules, "iptables -D FORWARD -i %i -j ACCEPT")
		rules = append(rules, fmt.Sprintf("iptables -t nat -D POSTROUTING -o %s -j MASQUERADE", c.OutInterface))
	}

	// Check if an IPv6 address is configured
	if c.IPv6Addr != "" {
		rules = append(rules, "ip6tables -D FORWARD -i %i -j ACCEPT")
		rules = append(rules, fmt.Sprintf("ip6tables -t nat -D POSTROUTING -o %s -j MASQUERADE", c.OutInterface))
	}

	// Return the generated rules as a semicolon-separated string
	return strings.Join(rules, "; ")
}

// PostUp generates the PostUp rules based on IPv4 and IPv6 settings
func (c *ServerConfig) PostUp() string {
	// Initialize an empty slice to store the rules
	var rules []string

	// Check if an IPv4 address is configured
	if c.IPv4Addr != "" {
		rules = append(rules, "iptables -A FORWARD -i %i -j ACCEPT")
		rules = append(rules, fmt.Sprintf("iptables -t nat -A POSTROUTING -o %s -j MASQUERADE", c.OutInterface))
	}

	// Check if an IPv6 address is configured
	if c.IPv6Addr != "" {
		rules = append(rules, "ip6tables -A FORWARD -i %i -j ACCEPT")
		rules = append(rules, fmt.Sprintf("ip6tables -t nat -A POSTROUTING -o %s -j MASQUERADE", c.OutInterface))
	}

	// Return the generated rules as a semicolon-separated string
	return strings.Join(rules, "; ")
}
