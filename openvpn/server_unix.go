//go:build darwin || linux

package openvpn

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return name
}
