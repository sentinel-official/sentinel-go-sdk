package wireguard

// deviceName returns the name of the WireGuard interface.
func (s *Server) deviceName() (string, error) {
	return s.name, nil
}
