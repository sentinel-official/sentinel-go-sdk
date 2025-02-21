//go:build darwin || linux

package wireguard

import (
	"fmt"
	"net/netip"
	"strings"
)

// PreDown generates the PreDown rules based on IPv4 and IPv6 settings
func (c *ClientConfig) PreDown() string {
	// Get the list of addresses and exclude addresses for the client
	addrs := c.GetAddrs()
	excludeAddrs := c.GetExcludeAddrs()

	// Helper function to generate the necessary iptables/ip6tables rules
	generateRules := func(execName string, comp func(netip.Addr) bool, matchRule string) (rules []string) {
		// Loop over all the addresses
		for _, v := range addrs {
			if comp(v.Addr()) {
				// Loop over all the exclude addresses
				for _, v := range excludeAddrs {
					if comp(v.Addr()) {
						// Remove an ACCEPT rule for the exclude address
						rules = append(rules, fmt.Sprintf("%s -D OUTPUT %s -d %s -j ACCEPT", execName, matchRule, v))
					}
				}
			}
		}

		// If any ACCEPT rules were generated, remove a default DROP rule to block further traffic
		if len(rules) != 0 {
			rules = append(rules, fmt.Sprintf("%s -D OUTPUT %s -j DROP", execName, matchRule))
		}

		// Return the generated rules
		return rules
	}

	// Define the matchRule for the firewall rules using the WireGuard interface name
	matchRule := fmt.Sprintf("! -o %s -m mark ! --mark $(wg show %s fwmark)", c.Name, c.Name)

	// Generate the rules for IPv4 (iptables) and IPv6 (ip6tables)
	rules4 := generateRules("iptables", func(v netip.Addr) bool { return v.Is4() }, matchRule)
	rules6 := generateRules("ip6tables", func(v netip.Addr) bool { return v.Is6() }, matchRule)

	// Return the combined IPv4 and IPv6 rules as a semicolon-separated string
	return strings.Join(append(rules4, rules6...), "; ")
}

// PostUp generates the PostUp rules based on IPv4 and IPv6 settings
func (c *ClientConfig) PostUp() string {
	// Get the list of addresses and exclude addresses for the client
	addrs := c.GetAddrs()
	excludeAddrs := c.GetExcludeAddrs()

	// Helper function to generate the necessary iptables/ip6tables rules
	generateRules := func(execName string, comp func(netip.Addr) bool, matchRule string) (rules []string) {
		// Loop over all the client addresses
		for _, v := range addrs {
			if comp(v.Addr()) {
				// Loop over all the exclude addresses
				for _, v := range excludeAddrs {
					if comp(v.Addr()) {
						// Add an ACCEPT rule for the exclude address
						rules = append(rules, fmt.Sprintf("%s -I OUTPUT %s -d %s -j ACCEPT", execName, matchRule, v))
					}
				}
			}
		}

		// If any ACCEPT rules were generated, add a default DROP rule to block further traffic
		if len(rules) != 0 {
			rules = append(rules, fmt.Sprintf("%s -I OUTPUT %s -j DROP", execName, matchRule))
		}

		// Reverse the rules to ensure the default DROP is last
		for i, j := 0, len(rules)-1; i < j; i, j = i+1, j-1 {
			rules[i], rules[j] = rules[j], rules[i]
		}

		// Return the generated rules
		return rules
	}

	// Define the matchRule for the firewall rules using the WireGuard interface name
	matchRule := fmt.Sprintf("! -o %s -m mark ! --mark $(wg show %s fwmark)", c.Name, c.Name)

	// Generate the rules for IPv4 (iptables) and IPv6 (ip6tables)
	rules4 := generateRules("iptables", func(v netip.Addr) bool { return v.Is4() }, matchRule)
	rules6 := generateRules("ip6tables", func(v netip.Addr) bool { return v.Is6() }, matchRule)

	// Return the combined IPv4 and IPv6 rules as a semicolon-separated string
	return strings.Join(append(rules4, rules6...), "; ")
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
