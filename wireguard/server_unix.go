//go:build darwin || linux

package wireguard

import (
	"fmt"
	"os/exec"
	"strings"
)

// execFile returns the name of the executable file.
func (s *Server) execFile(name string) string {
	return name
}

// Down shuts down the WireGuard interface.
func (s *Server) Down() error {
	// Executes the 'wg-quick down' command to bring down the interface.
	cfgFile := s.serviceConfigFilePath()
	cmd := exec.Command(
		s.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("down %s", cfgFile))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	return nil
}

// Up starts the WireGuard interface.
func (s *Server) Up() error {
	// Executes the 'wg-quick up' command to bring up the interface.
	cfgFile := s.serviceConfigFilePath()
	cmd := exec.CommandContext(
		s.ctx,
		s.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("up %s", cfgFile))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	return nil
}
