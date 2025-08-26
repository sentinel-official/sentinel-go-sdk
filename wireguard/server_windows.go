package wireguard

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// execFile returns the name of the executable file.
func (s *Server) execFile(name string) string {
	return ".\\" + filepath.Join("WireGuard", name+".exe")
}

// deviceName returns the name of the WireGuard interface.
func (s *Server) deviceName() (string, error) {
	return s.name, nil
}

// Down uninstalls the WireGuard tunnel service.
func (s *Server) Down() error {
	device, err := s.deviceName()
	if err != nil {
		return fmt.Errorf("failed to get device name: %w", err)
	}

	// Executes the command to uninstall the WireGuard tunnel service.
	cmd := exec.CommandContext(
		context.Background(),
		s.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", device))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}

// Up installs the WireGuard tunnel service.
func (s *Server) Up() error {
	// Executes the command to install the WireGuard tunnel service.
	cfgFile := s.serviceConfigFilePath()
	cmd := exec.CommandContext(
		s.ctx,
		s.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", cfgFile))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}
