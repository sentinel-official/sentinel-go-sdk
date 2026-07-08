package wireguard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sentinel-official/sentinel-go-sdk/libs/netip"
	"github.com/sentinel-official/sentinel-go-sdk/libs/safe"
	"github.com/sentinel-official/sentinel-go-sdk/process"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Server implements types.ServerService interface.
var _ types.ServerService = (*Server)(nil)

// Server represents the WireGuard server service instance.
type Server struct {
	*process.Manager // Embedded process manager for handling lifecycle.

	cfg     *ServerConfig // Configuration settings for the service.
	device  string        // Name of the WireGuard network interface.
	homeDir string        // Home directory of the service.

	metadata []*ServerMetadata       // Metadata containing server-specific details.
	peers    *safe.Map[string, Peer] // Thread-safe map to manage peers connected to the server.
	pools    *netip.AddrPoolSet      // Address pool set for allocating IP addresses to peers.
}

// NewServer creates a new Server instance.
func NewServer(name, appDir string, cfg *ServerConfig) *Server {
	return &Server{
		Manager: process.NewManager(name),
		cfg:     cfg,
		device:  defaultDevice,
		homeDir: filepath.Join(appDir, "wireguard"),
		peers:   safe.NewMap[string, Peer](),
	}
}

// WithDevice sets the WireGuard network interface name and returns the updated Server instance.
func (s *Server) WithDevice(device string) *Server {
	s.device = device

	return s
}

// Type returns the service type of the server.
func (s *Server) Type() types.ServiceType {
	return types.ServiceTypeWireGuard
}

// Metadata returns the service metadata of the server.
func (s *Server) Metadata() any { return s.metadata }

// IsRunning checks if the WireGuard interface is up and active.
func (s *Server) IsRunning() (bool, error) {
	// Executes the 'wg show' command to check the interface status.
	cmd := exec.CommandContext(
		context.Background(),
		s.execFile("wg"),
		"show", s.device,
	)

	// Capture stderr output.
	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	// Run the command and handle errors.
	if err := cmd.Run(); err != nil {
		// Treat a missing interface or absent kernel module (userspace fallback) as not running.
		if out := stderr.String(); strings.Contains(out, "No such device") || strings.Contains(out, "Protocol not supported") {
			return false, nil
		}

		return false, fmt.Errorf("running command: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return true, nil
}

// Init sets up the server configuration, creating directories and writing defaults unless config exists.
func (s *Server) Init(force bool) error {
	// Create the home directory if it doesn't exist
	if err := os.MkdirAll(s.homeDir, 0700); err != nil {
		return fmt.Errorf("creating home directory %q: %w", s.homeDir, err)
	}

	// Construct the full path to the config file
	cfgFile := s.appConfigFile()

	// Check if the config file exists at the specified path
	exists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
	}

	// Write config only if file doesn't exist or force flag is enabled
	if !exists || force {
		if err := s.cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("writing app config file %q: %w", cfgFile, err)
		}
	}

	return nil
}

// Setup prepares the WireGuard server service for operation.
func (s *Server) Setup(ctx context.Context) error {
	return s.Manager.Setup(ctx, func() error { //nolint:wrapcheck
		// Construct the full path to the config file
		cfgFile := s.appConfigFile()

		// Check if the config file exists at the specified path
		exists, err := utils.IsFileExists(cfgFile)
		if err != nil {
			return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
		}

		// If the config file exists, proceed to read its contents
		if exists {
			if err := s.cfg.ReadAppConfig(cfgFile); err != nil {
				return fmt.Errorf("reading app config file %q: %w", cfgFile, err)
			}
		}

		// Validate the config
		if err := s.cfg.Validate(); err != nil {
			return fmt.Errorf("validating config: %w", err)
		}

		// Write configuration to file.
		cfgFile = s.serviceConfigFile()
		if err := s.cfg.WriteServiceConfig(cfgFile); err != nil {
			return fmt.Errorf("writing service config file %q: %w", cfgFile, err)
		}

		// Initialize the addr pool set from the configuration.
		s.pools, err = s.cfg.AddrPoolSet()
		if err != nil {
			return fmt.Errorf("initializing addr pool set: %w", err)
		}

		// Set the server metadata.
		s.metadata = []*ServerMetadata{
			{
				Port:      s.cfg.OutPort(),
				PublicKey: s.cfg.PublicKey(),
			},
		}

		return nil
	})
}

// Start starts the WireGuard server service.
func (s *Server) Start(parent context.Context) (context.Context, error) {
	return s.Manager.Start(parent, func(ctx context.Context) error { //nolint:wrapcheck
		// Start the WireGuard process.
		cmd, err := s.startCmd(ctx)
		if err != nil {
			return fmt.Errorf("preparing command: %w", err)
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running command: %w", err)
		}

		// Periodically sync peer statistics.
		s.Go(ctx, func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
					// Check if service is up before syncing peers.
					ok, err := s.IsRunning() //nolint:contextcheck
					if err != nil {
						return fmt.Errorf("checking service status: %w", err)
					}

					if !ok {
						continue
					}

					// Sync peer statistics from WireGuard.
					if err := s.syncPeers(ctx); err != nil {
						return fmt.Errorf("syncing peer statistics: %w", err)
					}
				}
			}
		})

		return nil
	})
}

// Stop stops the WireGuard server service.
func (s *Server) Stop() error {
	return s.Manager.Stop(func() error { //nolint:wrapcheck
		cmd, err := s.stopCmd()
		if err != nil {
			return fmt.Errorf("preparing command: %w", err)
		}

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running command: %w", err)
		}

		return nil
	})
}

// Wait waits for all background goroutines to complete.
func (s *Server) Wait(ctx context.Context) error {
	return s.Manager.Wait(ctx, nil) //nolint:wrapcheck
}

// Cleanup removes service configuration files.
func (s *Server) Cleanup() error {
	return s.Manager.Cleanup(func() error { //nolint:wrapcheck
		// Removes configuration file.
		cfgFile := s.serviceConfigFile()
		if err := utils.RemoveFile(cfgFile); err != nil {
			return fmt.Errorf("removing service config file %q: %w", cfgFile, err)
		}

		return nil
	})
}

// AddPeer adds a new peer to the WireGuard server.
func (s *Server) AddPeer(ctx context.Context, req any) (string, any, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return "", nil, fmt.Errorf("parsing request: %w", err)
	}

	if err := r.Validate(); err != nil {
		return "", nil, fmt.Errorf("validating request: %w", err)
	}

	// Retrieve the identity from the request.
	id := r.ID()

	// Acquire addrs from the pool for the new peer.
	addrs, err := s.pools.Acquire()
	if err != nil {
		return "", nil, fmt.Errorf("acquiring peer %q addrs: %w", id, err)
	}

	// Ensure addresses are released if peer addition fails.
	defer func() {
		if ok := s.peers.Exists(id); !ok {
			if err := s.pools.Release(addrs); err != nil {
				panic(fmt.Errorf("releasing peer %q addrs %v: %w", id, addrs, err))
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
			return "", nil, fmt.Errorf("getting peer %q prefix addr from addr %q: %w", id, addr, err)
		}

		allowedIPs = append(allowedIPs, prefix.String())
		prefixAddrs = append(prefixAddrs, &netip.Prefix{Prefix: prefix})
	}

	// Executes the 'wg set' command to add the peer to the WireGuard interface.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		"set", s.device, "peer", id, "allowed-ips", strings.Join(allowedIPs, ","),
	)

	// Run the command and check for errors.
	if err := cmd.Run(); err != nil {
		return "", nil, fmt.Errorf("running command: %w", err)
	}

	// Save the peer details in the local peers map.
	now := time.Now()
	s.peers.Set(id, Peer{
		ID:       id,
		Addrs:    addrs,
		Current:  types.NewPeerStatistics(now),
		Previous: types.NewPeerStatistics(now),
	})

	return id, &AddPeerResponse{
		Addrs:    prefixAddrs,
		Metadata: s.metadata,
	}, nil
}

// HasPeer checks if a peer exists in the WireGuard server's peer list.
func (s *Server) HasPeer(_ context.Context, id string) (bool, error) {
	return s.peers.Exists(id), nil
}

// RemovePeer removes a peer from the WireGuard server.
func (s *Server) RemovePeer(ctx context.Context, id string) error {
	// Executes the 'wg set' command to remove the peer from the WireGuard interface.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		"set", s.device, "peer", id, "remove",
	)

	// Run the command and check for errors.
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	// Get the peer from the map.
	peer, ok := s.peers.Get(id)
	if !ok {
		return fmt.Errorf("peer %q does not exist", id)
	}

	// Release the addrs back to the pool.
	if err := s.pools.Release(peer.Addrs); err != nil {
		return fmt.Errorf("releasing peer %q addrs %v: %w", id, peer.Addrs, err)
	}

	// Remove the peer information from the local collection.
	s.peers.Delete(id, nil)

	return nil
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
	s.peers.RangeGet(func(_ string, v Peer) bool {
		items[v.ID] = &types.PeerStatistics{
			RxBytes:   v.TotalRxBytes(),
			TxBytes:   v.TotalTxBytes(),
			CreatedAt: v.CreatedAt(),
			UpdatedAt: v.UpdatedAt(),
		}

		return false
	})

	return items, nil
}

func (s *Server) appConfigFile() string     { return filepath.Join(s.homeDir, "config.toml") }
func (s *Server) serviceConfigFile() string { return filepath.Join(s.homeDir, s.device+".conf") }

// syncPeers retrieves the latest peer transfer statistics from WireGuard
// and updates the in-memory peer data accordingly.
func (s *Server) syncPeers(ctx context.Context) error {
	// Executes the 'wg show' command to get transfer statistics.
	cmd := exec.CommandContext(
		ctx,
		s.execFile("wg"),
		"show", s.device, "transfer",
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("running command: %w", err)
	}

	// Split the command output into lines and process each line.
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		columns := strings.Split(line, "\t")
		if len(columns) != 3 {
			continue
		}

		// Parse peer upload traffic stats.
		rxBytes, err := strconv.ParseInt(columns[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing peer %q rx bytes %q: %w", columns[0], columns[1], err)
		}

		// Parse peer download traffic stats.
		txBytes, err := strconv.ParseInt(columns[2], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing peer %q tx bytes %q: %w", columns[0], columns[2], err)
		}

		createdAt := time.Time{}
		now := time.Now()

		// Update peer statistics in thread-safe map.
		s.peers.Update(columns[0], func(v Peer, found bool) (Peer, bool) {
			if !found {
				return v, false
			}

			if createdAt.After(v.Current.CreatedAt) {
				v.Previous.RxBytes += v.Current.RxBytes
				v.Previous.TxBytes += v.Current.TxBytes
				v.Previous.UpdatedAt = now

				v.Current = types.NewPeerStatistics(createdAt)
			}

			if v.Current.RxBytes > 0 && rxBytes == v.Current.RxBytes {
				return v, false
			}

			v.Current.RxBytes = rxBytes
			v.Current.TxBytes = txBytes
			v.Current.UpdatedAt = now

			return v, true
		})
	}

	return nil
}
