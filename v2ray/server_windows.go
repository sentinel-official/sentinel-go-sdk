package v2ray

import (
	"path/filepath"
)

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return ".\\" + filepath.Join("V2Ray", name+".exe")
}
