//go:build darwin || linux

package hysteria2

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return name
}
