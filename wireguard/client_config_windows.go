package wireguard

// PostUp generates PostUp rules for IPv4 and IPv6 settings.
func (c *ClientConfig) PostUp() []string {
	return nil
}

// PreDown generates PreDown rules to remove the PostUp rules for IPv4 and IPv6.
func (c *ClientConfig) PreDown() []string {
	return nil
}
