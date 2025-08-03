package wireguard

// deviceName returns the name of the WireGuard interface.
func (c *Client) deviceName() (string, error) {
	return c.name, nil
}
