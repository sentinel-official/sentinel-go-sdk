package amneziawg

import (
	"context"
	"os/exec"
	"path/filepath"
)

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return ".\\" + filepath.Join("AmneziaWG", name+".exe")
}

// startCmd returns the command to bring up the AmneziaWG interface.
func (s *Server) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the command to install the AmneziaWG tunnel service.
	cfgFile := s.serviceConfigFile()
	cmd := exec.CommandContext(
		ctx,
		s.execFile("amneziawg"),
		"/installtunnelservice", cfgFile,
	)

	return cmd, nil
}

// stopCmd returns the command to bring down the AmneziaWG interface.
func (s *Server) stopCmd() (*exec.Cmd, error) {
	// Build the command to uninstall the AmneziaWG tunnel service.
	cmd := exec.CommandContext(
		context.Background(),
		s.execFile("amneziawg"),
		"/uninstalltunnelservice", s.device,
	)

	return cmd, nil
}
