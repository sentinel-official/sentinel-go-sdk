//go:build darwin || linux

package xray

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return name
}
