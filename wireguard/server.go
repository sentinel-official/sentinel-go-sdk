package wireguard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Server implements types.ServerService interface.
var _ types.ServerService = (*Server)(nil)

// Server represents the WireGuard server instance.
type Server struct {
	homeDir  string            // Home directory of the WireGuard server.
	metadata []*ServerMetadata // Metadata containing server-specific details.
	name     string            // Name of the server instance.
	pm       *PeerManager      // Peer manager for handling peer information.
}

// NewServer creates a new Server instance.
func NewServer(appDir string) *Server {
	return &Server{
		homeDir: filepath.Join(appDir, "wireguard"),
		name:    "wg0",
	}
}

// WithName sets the name for the server and returns the updated Server instance.
func (s *Server) WithName(name string) *Server {
	s.name = name
	return s
}

// appConfigFilePath returns the full path to the application's configuration file.
func (s *Server) appConfigFilePath() string {
	return filepath.Join(s.homeDir, "config.toml")
}

// serviceConfigFilePath returns the full path to the service-specific configuration file.
func (s *Server) serviceConfigFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.conf", s.name))
}

// Type returns the service type of the server.
func (s *Server) Type() types.ServiceType {
	return types.ServiceTypeWireGuard
}

// Init sets up service configuration, creating directories and writing defaults unless config exists.
func (s *Server) Init(force bool) error {
	// Create the home directory if it doesn't exist
	if err := os.MkdirAll(s.homeDir, 0755); err != nil {
		return fmt.Errorf("failed to create home directory: %w", err)
	}

	// Construct the full path to the config file
	cfgFile := s.appConfigFilePath()

	// Check if the config file exists at the specified path
	cfgFileExists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}

	// Write default config only if file doesn't exist or force flag is enabled
	if !cfgFileExists || force {
		cfg := DefaultServerConfig()
		if err := cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	return nil
}

// IsUp checks if the WireGuard server process is running.
func (s *Server) IsUp(ctx context.Context) (bool, error) {
	// Retrieves the interface name.
	iface, err := s.interfaceName()
	if err != nil {
		return false, fmt.Errorf("failed to get interface name: %w", err)
	}

	// Executes the 'wg show' command to check the interface status.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		strings.Fields(fmt.Sprintf("show %s", iface))...,
	)

	// Capture stderr output.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command and handle errors.
	if err := cmd.Run(); err != nil {
		// Check if the error matches "No such device".
		if strings.Contains(stderr.String(), "No such device") {
			return false, nil
		}

		return false, fmt.Errorf("failed to run command: %w", err)
	}

	return true, nil
}

// PreUp writes the configuration to the config file before starting the server process.
func (s *Server) PreUp(_ interface{}) error {
	// Initialize viper instance
	v := viper.New()

	// Construct the full path to the config file
	cfgFile := s.appConfigFilePath()

	// Check if the config file exists at the specified path
	cfgFileExists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}

	// If the config file exists, proceed to read its contents
	if cfgFileExists {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal configuration into the config object
	cfg := DefaultServerConfig()
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to validate config file: %w", err)
	}

	pools, err := cfg.IPPools()
	if err != nil {
		return fmt.Errorf("failed to get ip pools: %w", err)
	}

	s.pm = NewPeerManager(pools...)
	s.name = cfg.InInterface
	s.metadata = []*ServerMetadata{
		{
			Port:      cfg.OutPort(),
			PublicKey: cfg.PublicKey(),
		},
	}

	// Writes configuration to file.
	cfgFile = s.serviceConfigFilePath()
	if err := cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// PostUp performs operations after the server process is started.
func (s *Server) PostUp() error {
	return nil
}

// PreDown performs operations before the server process is terminated.
func (s *Server) PreDown() error {
	return nil
}

// PostDown performs cleanup operations after the server process is terminated.
func (s *Server) PostDown() error {
	// Removes configuration file.
	cfgFile := s.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("failed to remove config: %w", err)
	}

	return nil
}

// AddPeer adds a new peer to the WireGuard server.
func (s *Server) AddPeer(ctx context.Context, req interface{}) (res interface{}, err error) {
	// Parse the request to ServiceRequest type.
	r, err := parseServiceRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Retrieve the identity from the request.
	identity := r.PublicKey.String()

	// Add peer to the peer manager and retrieve assigned IP addresses.
	addrs, err := s.pm.Put(identity)
	if err != nil {
		return nil, fmt.Errorf("failed to put peer: %w", err)
	}
	if len(addrs) == 0 {
		return nil, errors.New("no addrs available")
	}

	var allowedIPs []string
	for _, addr := range addrs {
		allowedIPs = append(allowedIPs, addr.String())
	}

	// Executes the 'wg set' command to add the peer to the WireGuard interface.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		strings.Fields(fmt.Sprintf("set %s peer %s allowed-ips %s", s.name, identity, strings.Join(allowedIPs, ",")))...,
	)

	// Run the command and check for errors.
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run command: %w", err)
	}

	return &AddPeerResponse{
		Addrs:    addrs,
		Metadata: s.metadata,
	}, nil
}

// HasPeer checks if a peer exists in the WireGuard server's peer list.
func (s *Server) HasPeer(_ context.Context, req interface{}) (bool, error) {
	// Parse the request to ServiceRequest type.
	r, err := parseServiceRequest(req)
	if err != nil {
		return false, fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return false, fmt.Errorf("invalid request: %w", err)
	}

	// Retrieve the identity from the request.
	identity := r.PublicKey.String()
	peer := s.pm.Get(identity)

	// Return true if the peer exists, otherwise false.
	return peer != nil, nil
}

// RemovePeer removes a peer from the WireGuard server.
func (s *Server) RemovePeer(ctx context.Context, req interface{}) error {
	// Parse the request to ServiceRequest type.
	r, err := parseServiceRequest(req)
	if err != nil {
		return fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// Retrieve the identity from the request.
	identity := r.PublicKey.String()

	// Executes the 'wg set' command to remove the peer from the WireGuard interface.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		strings.Fields(fmt.Sprintf(`set %s peer %s remove`, s.name, identity))...,
	)

	// Run the command and check for errors.
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	// Remove the peer information from the local collection.
	s.pm.Delete(identity)
	return nil
}

// PeerCount returns the number of peers connected to the WireGuard server.
func (s *Server) PeerCount() int {
	return s.pm.Len()
}

// PeerStatistics retrieves statistics for each peer connected to the WireGuard server.
func (s *Server) PeerStatistics(ctx context.Context) (items []*types.PeerStatistic, err error) {
	// Retrieves the interface name.
	iface, err := s.interfaceName()
	if err != nil {
		return nil, fmt.Errorf("failed to get interface name: %w", err)
	}

	// Executes the 'wg show' command to get transfer statistics.
	output, err := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		strings.Fields(fmt.Sprintf("show %s transfer", iface))...,
	).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run command: %w", err)
	}

	// Split the command output into lines and process each line.
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		columns := strings.Split(line, "\t")
		if len(columns) != 3 {
			continue
		}

		// Parse upload traffic stats.
		uploadBytes, err := strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse upload bytes: %w", err)
		}

		// Parse download traffic stats.
		downloadBytes, err := strconv.ParseInt(columns[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse download bytes: %w", err)
		}

		// Append peer statistics to the result collection.
		items = append(
			items,
			&types.PeerStatistic{
				Key:           columns[0],
				DownloadBytes: downloadBytes,
				UploadBytes:   uploadBytes,
			},
		)
	}

	// Return the constructed collection of peer statistics.
	return items, nil
}
