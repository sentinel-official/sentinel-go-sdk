//go:build darwin || linux

package v2ray

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return name
}
