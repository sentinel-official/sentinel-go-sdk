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
	return ".\\" + filepath.Join("WireGuard", s.name+".exe")
}

// interfaceName returns the name of the WireGuard interface.
func (s *Server) interfaceName() (string, error) {
	return s.name, nil
}

// Down uninstalls the WireGuard tunnel service.
func (s *Server) Down(ctx context.Context) error {
	iface, err := s.interfaceName()
	if err != nil {
		return fmt.Errorf("failed to get interface name: %w", err)
	}

	// Executes the command to uninstall the WireGuard tunnel service.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", iface))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}

// Up installs the WireGuard tunnel service.
func (s *Server) Up(ctx context.Context) error {
	// Executes the command to install the WireGuard tunnel service.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wireguard"),
		strings.Fields(fmt.Sprintf("/uninstalltunnelservice %s", s.configFilePath()))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}
