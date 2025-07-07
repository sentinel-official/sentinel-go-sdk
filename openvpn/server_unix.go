//go:build darwin || linux

package openvpn

// execFile returns the name of the executable file for the OpenVPN server.
func (s *Server) execFile(name string) string {
	return name
}
