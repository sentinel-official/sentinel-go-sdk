//go:build darwin || linux

package wireguard

import (
	"context"
	"os/exec"
	"syscall"
)

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return name
}

// startCmd returns the command to bring up the WireGuard interface.
func (s *Server) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the 'wg-quick up' command with the service config file.
	cfgFile := s.serviceConfigFile()
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg-quick"),
		"up", cfgFile,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd, nil
}

// stopCmd returns the command to bring down the WireGuard interface.
func (s *Server) stopCmd() (*exec.Cmd, error) {
	// Build the 'wg-quick down' command with the service config file.
	cfgFile := s.serviceConfigFile()
	cmd := exec.CommandContext(
		context.Background(),
		s.execFile("wg-quick"),
		"down", cfgFile,
	)

	return cmd, nil
}
