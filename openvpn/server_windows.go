package openvpn

import (
	"path/filepath"
)

// execFile returns the name of the executable file for the OpenVPN server.
func (s *Server) execFile(name string) string {
	return ".\\" + filepath.Join("OpenVPN", name+".exe")
}
