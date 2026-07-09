package xray

import (
	"path/filepath"
)

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return ".\\" + filepath.Join("Xray", name+".exe")
}
