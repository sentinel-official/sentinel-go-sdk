//go:build darwin || linux

package openvpn

import (
	"fmt"
	"strings"
)

// Down generates the Down rules based on IPv4 and IPv6 settings.
func (c *ServerConfig) Down() string {
	var rules []string

	// Check if an IPv4 address is configured
	if c.IPv4Addr != "" {
		rules = append(rules, fmt.Sprintf("iptables -D FORWARD -i $dev -o %s -j ACCEPT", c.OutInterface))
		rules = append(rules, fmt.Sprintf("iptables -t nat -D POSTROUTING -o %s -j MASQUERADE", c.OutInterface))
	}

	// Check if an IPv6 address is configured
	if c.IPv6Addr != "" {
		rules = append(rules, fmt.Sprintf("ip6tables -D FORWARD -i $dev -o %s -j ACCEPT", c.OutInterface))
		rules = append(rules, fmt.Sprintf("ip6tables -t nat -D POSTROUTING -o %s -j MASQUERADE", c.OutInterface))
	}

	return strings.Join(rules, "; ")
}

// Up generates the Up rules based on IPv4 and IPv6 settings.
func (c *ServerConfig) Up() string {
	var rules []string

	// Check if an IPv4 address is configured
	if c.IPv4Addr != "" {
		rules = append(rules, fmt.Sprintf("iptables -A FORWARD -i $dev -o %s -j ACCEPT", c.OutInterface))
		rules = append(rules, fmt.Sprintf("iptables -t nat -A POSTROUTING -o %s -j MASQUERADE", c.OutInterface))
	}

	// Check if an IPv6 address is configured
	if c.IPv6Addr != "" {
		rules = append(rules, fmt.Sprintf("ip6tables -A FORWARD -i $dev -o %s -j ACCEPT", c.OutInterface))
		rules = append(rules, fmt.Sprintf("ip6tables -t nat -A POSTROUTING -o %s -j MASQUERADE", c.OutInterface))
	}

	return strings.Join(rules, "; ")
}
