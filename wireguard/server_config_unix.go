//go:build darwin || linux

package wireguard

import (
	"fmt"
)

// PostDown generates the PostDown rules based on IPv4 and IPv6 settings
func (c *ServerConfig) PostDown() (rules []string) {
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

	return rules
}

// PostUp generates the PostUp rules based on IPv4 and IPv6 settings
func (c *ServerConfig) PostUp() (rules []string) {
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

	return rules
}
