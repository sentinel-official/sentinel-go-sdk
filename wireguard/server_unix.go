//go:build darwin || linux

package wireguard

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
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
		strings.Fields(fmt.Sprintf("up %s", cfgFile))...,
	)

	return cmd, nil
}

// stopCmd returns the command to bring down the WireGuard interface.
func (s *Server) stopCmd() (*exec.Cmd, error) {
	// Build the 'wg-quick down' command with the service config file.
	cfgFile := s.serviceConfigFile()
	cmd := exec.Command(
		s.execFile("wg-quick"),
		strings.Fields(fmt.Sprintf("down %s", cfgFile))...,
	)

	return cmd, nil
}
