//go:build darwin || linux

package wireguard

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// execFile returns the name of the executable file.
func (s *Server) execFile(name string) string {
	return name
}

// Down shuts down the WireGuard interface.
func (s *Server) Down(ctx context.Context) error {
	// Executes the 'wg-quick down' command to bring down the interface.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("down %s", s.configFilePath()))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}

// Up starts the WireGuard interface.
func (s *Server) Up(ctx context.Context) error {
	// Executes the 'wg-quick up' command to bring up the interface.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("up %s", s.configFilePath()))...,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	return nil
}
