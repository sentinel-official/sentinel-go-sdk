//go:build darwin || linux

package amneziawg

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

// execFile returns the executable name.
func (s *Server) execFile(name string) string {
	return name
}

// startCmd returns the command to bring up the AmneziaWG interface.
func (s *Server) startCmd(ctx context.Context) (*exec.Cmd, error) {
	// Build the 'awg-quick up' command with the service config file.
	cfgFile := s.serviceConfigFile()
	cmd := exec.CommandContext(
		ctx,
		s.execFile("awg-quick"),
		"up", cfgFile,
	)
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd, nil
}

// stopCmd returns the command to bring down the AmneziaWG interface.
func (s *Server) stopCmd() (*exec.Cmd, error) {
	// Build the 'awg-quick down' command with the service config file.
	cfgFile := s.serviceConfigFile()
	cmd := exec.CommandContext(
		context.Background(),
		s.execFile("awg-quick"),
		"down", cfgFile,
	)
	cmd.Stderr = os.Stderr

	return cmd, nil
}
