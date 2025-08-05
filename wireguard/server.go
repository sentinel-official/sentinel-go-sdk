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
	"time"

	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/libs/safe"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Server implements types.ServerService interface.
var _ types.ServerService = (*Server)(nil)

// Server represents the WireGuard server instance.
type Server struct {
	homeDir  string                  // Home directory of the WireGuard server.
	metadata []*ServerMetadata       // Metadata containing server-specific details.
	name     string                  // Name of the server instance.
	peers    *safe.Map[string, Peer] // Thread-safe map to manage peers connected to the server.
	pools    *netip.AddrPoolSet      // Address pool set for allocating IP addresses to peers.

	cancel context.CancelFunc // Context cancel function to stop background tasks.
	ctx    context.Context    // Context for server lifecycle management.
	eg     *errgroup.Group    // Error group for managing background goroutines.
}

// NewServer creates a new Server instance.
func NewServer(appDir string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Server{
		homeDir: filepath.Join(appDir, "wireguard"),
		name:    "wg0",
		peers:   safe.NewMap[string, Peer](),
		cancel:  cancel,
		ctx:     ctx,
		eg:      eg,
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
		return fmt.Errorf("failed to check config file existence: %w", err)
	}

	// Write default config only if file doesn't exist or force flag is enabled
	if !cfgFileExists || force {
		cfg := DefaultServerConfig()
		if err := cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("failed to write app config: %w", err)
		}
	}

	return nil
}

// IsUp checks if the WireGuard server process is running.
func (s *Server) IsUp() (bool, error) {
	// Retrieves the device name.
	device, err := s.deviceName()
	if err != nil {
		return false, fmt.Errorf("failed to get device name: %w", err)
	}
	if device == "" {
		return false, nil
	}

	// Executes the 'wg show' command to check the interface status.
	cmd := exec.CommandContext(
		context.Background(),
		s.execFile("wg"),
		strings.Fields(fmt.Sprintf("show %s", device))...,
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

// PreUp performs initialization tasks before starting the WireGuard service.
func (s *Server) PreUp(_ context.Context) error {
	// Initialize viper instance
	v := viper.New()

	// Construct the full path to the config file
	cfgFile := s.appConfigFilePath()

	// Check if the config file exists at the specified path
	cfgFileExists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to check config file existence: %w", err)
	}

	// If the config file exists, proceed to read its contents
	if cfgFileExists {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read app config: %w", err)
		}
	}

	// Unmarshal configuration into the config object
	cfg := DefaultServerConfig()
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal app config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid app config: %w", err)
	}

	// Initialize the addr pool set from the configuration.
	s.pools, err = cfg.AddrPoolSet()
	if err != nil {
		return fmt.Errorf("failed to initialize addr pool set: %w", err)
	}

	// Set the server metadata.
	s.metadata = []*ServerMetadata{
		{
			Port:      cfg.OutPort(),
			PublicKey: cfg.PublicKey(),
		},
	}

	// Writes configuration to file.
	cfgFile = s.serviceConfigFilePath()
	if err := cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("failed to write service config: %w", err)
	}

	return nil
}

// PostUp starts background peer synchronization after the server is up.
func (s *Server) PostUp(ctx context.Context) error {
	// Create a context that cancels if either server or input context is done.
	ctx, _ = utils.AnyDoneContext(s.ctx, ctx)

	// Start background goroutine for periodic peer statistics updates.
	s.eg.Go(func() error {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// Blocking loop that runs until stop signal is received
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				// Check if server is up before syncing peers.
				ok, err := s.IsUp()
				if err != nil {
					return fmt.Errorf("failed to check status: %w", err)
				}
				if !ok {
					continue
				}

				// Sync peer statistics from WireGuard.
				if err := s.syncPeers(ctx); err != nil {
					return fmt.Errorf("failed to sync peer statistics: %w", err)
				}
			}
		}
	})

	return nil
}

// Wait waits for all background goroutines to complete.
func (s *Server) Wait() error {
	if err := s.eg.Wait(); err != nil {
		return err
	}

	return nil
}

// PreDown performs operations before the server process is terminated.
func (s *Server) PreDown() error {
	// Cancel background tasks if any.
	s.cancel()

	return nil
}

// PostDown cleans up configuration files after the server is stopped.
func (s *Server) PostDown() error {
	// Removes configuration file.
	cfgFile := s.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("failed to remove service config: %w", err)
	}

	return nil
}

// AddPeer adds a new peer to the WireGuard server.
func (s *Server) AddPeer(ctx context.Context, req interface{}) (string, interface{}, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse peer request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return "", nil, fmt.Errorf("invalid peer request: %w", err)
	}

	// Retrieve the identity from the request.
	id := r.ID()

	// Acquire addrs from the pool for the new peer.
	addrs, err := s.pools.Acquire()
	if err != nil {
		return "", nil, fmt.Errorf("failed to acquire addrs: %w", err)
	}

	// Ensure addresses are released if peer addition fails.
	defer func() {
		if ok := s.peers.Exists(id); !ok {
			if err := s.pools.Release(addrs); err != nil {
				panic(fmt.Errorf("failed to release addrs: %w", err))
			}
		}
	}()

	// Build allowed IPs and prefix addresses.
	allowedIPs := make([]string, 0, len(addrs))
	prefixAddrs := make([]*netip.Prefix, 0, len(addrs))

	for _, addr := range addrs {
		b := 32
		if addr.Is6() {
			b = 128
		}

		// Create IP prefixes for allowed IPs.
		prefix, err := addr.Prefix(b)
		if err != nil {
			return "", nil, fmt.Errorf("failed to generate prefix addr: %w", err)
		}

		allowedIPs = append(allowedIPs, prefix.String())
		prefixAddrs = append(prefixAddrs, &netip.Prefix{Prefix: prefix})
	}

	// Executes the 'wg set' command to add the peer to the WireGuard interface.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		strings.Fields(fmt.Sprintf("set %s peer %s allowed-ips %s", s.name, id, strings.Join(allowedIPs, ",")))...,
	)

	// Run the command and check for errors.
	if err := cmd.Run(); err != nil {
		return "", nil, fmt.Errorf("failed to run command: %w", err)
	}

	// Save the peer details in the local peers map.
	s.peers.Set(id, Peer{
		ID:        id,
		Addrs:     addrs,
		Current:   &types.PeerStatistics{},
		Previous:  &types.PeerStatistics{},
		Timestamp: time.Now(),
	})

	return id, &AddPeerResponse{
		Addrs:    prefixAddrs,
		Metadata: s.metadata,
	}, nil
}

// HasPeer checks if a peer exists in the WireGuard server's peer list.
func (s *Server) HasPeer(_ context.Context, req interface{}) (bool, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return false, fmt.Errorf("failed to parse peer request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return false, fmt.Errorf("invalid peer request: %w", err)
	}

	// Retrieve the identity from the request.
	id := r.ID()
	ok := s.peers.Exists(id)

	return ok, nil
}

// RemovePeer removes a peer from the WireGuard server.
func (s *Server) RemovePeer(ctx context.Context, req interface{}) (string, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return "", fmt.Errorf("failed to parse peer request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return "", fmt.Errorf("invalid peer request: %w", err)
	}

	// Retrieve the identity from the request.
	id := r.ID()

	// Executes the 'wg set' command to remove the peer from the WireGuard interface.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		strings.Fields(fmt.Sprintf(`set %s peer %s remove`, s.name, id))...,
	)

	// Run the command and check for errors.
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run command: %w", err)
	}

	// Get the peer from the map.
	peer, ok := s.peers.Get(id)
	if !ok {
		return "", errors.New("peer not found")
	}

	// Release the addrs back to the pool.
	if err := s.pools.Release(peer.Addrs); err != nil {
		return "", fmt.Errorf("failed to release addrs: %w", err)
	}

	// Remove the peer information from the local collection.
	s.peers.Delete(id)
	return id, nil
}

// PeersLen returns the number of peers connected to the WireGuard server.
func (s *Server) PeersLen() int {
	return s.peers.Len()
}

// PeerStatistics retrieves statistics for each peer connected to the WireGuard server.
func (s *Server) PeerStatistics() (map[string]*types.PeerStatistics, error) {
	// Create map to store statistics.
	items := make(map[string]*types.PeerStatistics)

	// Iterate over all peers and gather statistics.
	fn := func(_ string, peer Peer) (bool, error) {
		items[peer.ID] = &types.PeerStatistics{
			Duration: peer.TotalDuration(),
			RxBytes:  peer.TotalRxBytes(),
			TxBytes:  peer.TotalTxBytes(),
		}

		return false, nil
	}

	if err := s.peers.Range(fn); err != nil {
		return nil, fmt.Errorf("failed to range peer statistics: %w", err)
	}

	return items, nil
}

// syncPeers retrieves the latest peer transfer statistics from WireGuard
// and updates the in-memory peer data accordingly.
func (s *Server) syncPeers(ctx context.Context) error {
	// Retrieves the device name.
	device, err := s.deviceName()
	if err != nil {
		return fmt.Errorf("failed to get device name: %w", err)
	}
	if device == "" {
		return errors.New("empty device name")
	}

	// Executes the 'wg show' command to get transfer statistics.
	output, err := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		strings.Fields(fmt.Sprintf("show %s transfer", device))...,
	).Output()
	if err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}

	// Split the command output into lines and process each line.
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		columns := strings.Split(line, "\t")
		if len(columns) != 3 {
			continue
		}

		// Parse upload traffic stats.
		rxBytes, err := strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse download bytes: %w", err)
		}

		// Parse download traffic stats.
		txBytes, err := strconv.ParseInt(columns[2], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse upload bytes: %w", err)
		}

		// Update peer statistics in thread-safe map.
		s.peers.Update(columns[0], func(p Peer, ok bool) Peer {
			if !ok {
				return p
			}

			p.Current.RxBytes = rxBytes
			p.Current.TxBytes = txBytes
			p.Current.Duration = time.Since(p.Timestamp)

			return p
		})
	}

	return nil
}
