package hysteria2

import (
	"path/filepath"
)

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return ".\\" + filepath.Join("Hysteria2", name+".exe")
}
