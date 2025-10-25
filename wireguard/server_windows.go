package wireguard

import (
	"context"
	"os/exec"
	"path/filepath"
)

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return ".\\" + filepath.Join("WireGuard", name+".exe")
}

// startCmd returns the command to bring up the WireGuard interface.
func (s *Server) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the command to install the WireGuard tunnel service.
	cfgFile := s.serviceConfigFile()
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wireguard"),
		"/installtunnelservice", cfgFile,
	)

	return cmd, nil
}

// stopCmd returns the command to bring down the WireGuard interface.
func (s *Server) stopCmd() (*exec.Cmd, error) {
	// Build the command to uninstall the WireGuard tunnel service.
	cmd := exec.CommandContext(
		context.Background(),
		s.execFile("wireguard"),
		"/uninstalltunnelservice", c.device,
	)

	return cmd, nil
}
