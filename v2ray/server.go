package v2ray

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/spf13/viper"
	proxymancommand "github.com/v2fly/v2ray-core/v5/app/proxyman/command"
	statscommand "github.com/v2fly/v2ray-core/v5/app/stats/command"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sentinel-official/sentinel-go-sdk/libs/safe"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Server implements types.ServerService interface.
var _ types.ServerService = (*Server)(nil)

// Server represents the V2Ray server instance.
type Server struct {
	cmd      *exec.Cmd               // Command to run the V2Ray server.
	homeDir  string                  // Home directory of the V2Ray server.
	metadata []*ServerMetadata       // Metadata for server's inbound connections.
	name     string                  // Name of the server instance.
	peers    *safe.Map[string, Peer] // Peer manager for handling peer information.

	cancel context.CancelFunc // Context cancel function to stop background tasks.
	ctx    context.Context    // Context for server lifecycle management.
	eg     *errgroup.Group    // Error group for managing background goroutines.
}

// NewServer creates a new Server instance.
func NewServer(appDir string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Server{
		homeDir: filepath.Join(appDir, "v2ray"),
		name:    "v2ray",
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
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.json", s.name))
}

// pidFilePath returns the file path of the server's PID file.
func (s *Server) pidFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.pid", s.name))
}

// readPIDFromFile reads the PID from the server's PID file.
func (s *Server) readPIDFromFile() (int32, error) {
	// Get the full path to the PID file
	pidFile := s.pidFilePath()

	// Check if the PID file exists
	exists, err := utils.IsFileExists(pidFile)
	if err != nil {
		return 0, fmt.Errorf("failed to check existence of pid file: %w", err)
	}
	if !exists {
		return 0, nil
	}

	// Read PID from the PID file.
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	// Convert PID data to integer.
	pid, err := strconv.ParseInt(string(data), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse pid: %w", err)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("invalid pid %d", pid)
	}

	return int32(pid), nil
}

// writePIDToFile writes the given PID to the server's PID file.
func (s *Server) writePIDToFile(pid int) error {
	// Convert PID to byte slice.
	data := []byte(strconv.Itoa(pid))

	// Write PID to file with appropriate permissions.
	pidFile := s.pidFilePath()
	if err := os.WriteFile(pidFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// clientConn establishes a gRPC client connection to the V2Ray server.
func (s *Server) clientConn() (*grpc.ClientConn, error) {
	// Define the target address for the gRPC client connection.
	target := "127.0.0.1:2323"

	// Establish a gRPC client connection with specified options:
	// - WithTransportCredentials: Configures insecure transport credentials for the connection.
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	return conn, nil
}

// handlerServiceClient establishes a gRPC client connection to the V2Ray server's handler service.
func (s *Server) handlerServiceClient() (*grpc.ClientConn, proxymancommand.HandlerServiceClient, error) {
	// Establish a gRPC client connection using the clientConn method.
	conn, err := s.clientConn()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get grpc client connection: %w", err)
	}

	// Create a new HandlerServiceClient using the established connection.
	client := proxymancommand.NewHandlerServiceClient(conn)

	// Return both the connection and the client.
	return conn, client, nil
}

// statsServiceClient establishes a gRPC client connection to the V2Ray server's stats service.
func (s *Server) statsServiceClient() (*grpc.ClientConn, statscommand.StatsServiceClient, error) {
	// Establish a gRPC client connection using the clientConn method.
	conn, err := s.clientConn()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get grpc client connection: %w", err)
	}

	// Create a new StatsServiceClient using the established connection.
	client := statscommand.NewStatsServiceClient(conn)

	// Return both the connection and the client.
	return conn, client, nil
}

// Type returns the service type of the server.
func (s *Server) Type() types.ServiceType {
	return types.ServiceTypeV2Ray
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

// IsUp checks if the V2Ray server process is running.
func (s *Server) IsUp() (bool, error) {
	// Read PID from file.
	pid, err := s.readPIDFromFile()
	if err != nil {
		return false, fmt.Errorf("failed to read pid from file: %w", err)
	}
	if pid == 0 {
		return false, nil
	}

	// Retrieve process with the given PID.
	proc, err := process.NewProcess(pid)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get process: %w", err)
	}

	// Check if the process is running.
	ok, err := proc.IsRunning()
	if err != nil {
		return false, fmt.Errorf("failed to check running process: %w", err)
	}
	if !ok {
		return false, nil
	}

	// Retrieve the name of the process.
	name, err := proc.Name()
	if err != nil {
		return false, fmt.Errorf("failed to get process name: %w", err)
	}

	// Check if the process name matches constant v2ray.
	if name != v2ray {
		return false, nil
	}

	return true, nil
}

// PreUp writes the configuration to the config file before starting the server process.
func (s *Server) PreUp(_ context.Context) error {
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

	for _, inbound := range cfg.Inbounds {
		metadata := &ServerMetadata{
			Tag: inbound.Tag(),
		}

		s.metadata = append(s.metadata, metadata)
	}

	// Write configuration to file.
	cfgFile = s.serviceConfigFilePath()
	if err := cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("failed to write config to file: %w", err)
	}

	return nil
}

// Up starts the V2Ray server process.
func (s *Server) Up(ctx context.Context) error {
	// Create a context that cancels if either server or input context is done.
	ctx, _ = utils.AnyDoneContext(s.ctx, ctx)

	// Constructs the command to start the V2Ray server.
	cfgFile := s.serviceConfigFilePath()
	s.cmd = exec.CommandContext(
		ctx,
		s.execFile(v2ray),
		strings.Fields(fmt.Sprintf("run --config %s", cfgFile))...,
	)

	// Starts the V2Ray server process.
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for the V2Ray process to finish in a separate goroutine.
	s.eg.Go(func() error {
		if err := s.cmd.Wait(); err != nil {
			return fmt.Errorf("failed to wait command: %w", err)
		}

		return nil
	})

	return nil
}

// PostUp performs operations after the server process is started.
func (s *Server) PostUp(ctx context.Context) error {
	// Create a context that cancels if either server or input context is done.
	ctx, _ = utils.AnyDoneContext(s.ctx, ctx)

	// Write PID to file.
	if err := s.writePIDToFile(s.cmd.Process.Pid); err != nil {
		return fmt.Errorf("failed to write pid to file: %w", err)
	}

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

// Wait blocks until all goroutines in the error group finish or one returns an error.
func (s *Server) Wait() error {
	if err := s.eg.Wait(); err != nil {
		return err
	}

	return nil
}

// PreDown performs cleanup tasks before the server process is stopped.
// It cancels the server context to gracefully stop background operations.
func (s *Server) PreDown() error {
	// Cancel background tasks if any.
	s.cancel()

	return nil
}

// Down terminates the V2Ray server process.
func (s *Server) Down() error {
	// Read PID from file.
	pid, err := s.readPIDFromFile()
	if err != nil {
		return fmt.Errorf("failed to read pid from file: %w", err)
	}
	if pid == 0 {
		return nil
	}

	// Retrieve process with the given PID.
	proc, err := process.NewProcess(pid)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil
		}

		return fmt.Errorf("failed to get process: %w", err)
	}

	// Terminate the process.
	if err := proc.Terminate(); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	return nil
}

// PostDown performs cleanup operations after the server process is terminated.
func (s *Server) PostDown() error {
	// Removes configuration file.
	cfgFile := s.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	// Remove PID file.
	pidFile := s.pidFilePath()
	if err := utils.RemoveFile(pidFile); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

// AddPeer adds a new peer to the V2Ray server.
func (s *Server) AddPeer(ctx context.Context, req interface{}) (string, interface{}, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return "", nil, fmt.Errorf("invalid request: %w", err)
	}

	// Establish a gRPC client connection to the handler service.
	conn, client, err := s.handlerServiceClient()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get handler service client: %w", err)
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Retrieve the identity from the request.
	id := r.ID()

	for _, md := range s.metadata {
		// Prepare gRPC request to add a new user to the handler.
		in := &proxymancommand.AlterInboundRequest{
			Tag: md.Tag.String(),
			Operation: serial.ToTypedMessage(
				&proxymancommand.AddUserOperation{
					User: &protocol.User{
						Email:   id,
						Account: md.Tag.Account(r.UUID),
					},
				},
			),
		}

		// Send the request to add a user to the handler.
		if _, err := client.AlterInbound(ctx, in); err != nil {
			return "", nil, fmt.Errorf("failed to alter inbound: %w", err)
		}
	}

	// Save the peer details in the local peers map.
	s.peers.Set(id, Peer{
		ID: id,
	})

	// Return nil for success (no additional data to return in response).
	return id, &AddPeerResponse{
		Metadata: s.metadata,
	}, nil
}

// HasPeer checks if a peer exists in the V2Ray server's peer list.
func (s *Server) HasPeer(_ context.Context, req interface{}) (bool, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return false, fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return false, fmt.Errorf("invalid request: %w", err)
	}

	// Retrieve the identity from the request.
	id := r.ID()
	ok := s.peers.Exists(id)

	return ok, nil
}

// RemovePeer removes a peer from the V2Ray server.
func (s *Server) RemovePeer(ctx context.Context, req interface{}) (string, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return "", fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	// Establish a gRPC client connection to the handler service.
	conn, client, err := s.handlerServiceClient()
	if err != nil {
		return "", fmt.Errorf("failed to get handler service client: %w", err)
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Retrieve the identity from the request.
	id := r.ID()

	for _, md := range s.metadata {
		// Prepare gRPC request to remove a user from the handler.
		in := &proxymancommand.AlterInboundRequest{
			Tag: md.Tag.String(),
			Operation: serial.ToTypedMessage(
				&proxymancommand.RemoveUserOperation{
					Email: id,
				},
			),
		}

		// Send the request to remove a user from the handler.
		if _, err := client.AlterInbound(ctx, in); err != nil {
			// If the user is not found, continue without error.
			if !strings.Contains(err.Error(), "not found") {
				return "", fmt.Errorf("failed to alter inbound: %w", err)
			}
		}
	}

	// Remove the peer information from the local collection.
	s.peers.Delete(id)
	return id, nil
}

// PeersLen returns the number of peers connected to the V2Ray server.
func (s *Server) PeersLen() int {
	return s.peers.Len()
}

// PeerStatistics retrieves statistics for each peer connected to the V2Ray server.
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

// syncPeers retrieves the latest peer transfer statistics from the stats service
// and updates the in-memory peer data accordingly.
func (s *Server) syncPeers(ctx context.Context) error {
	// Establish a gRPC client connection to the stats service.
	conn, client, err := s.statsServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get stats service client: %w", err)
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Create a copy of the current peers to iterate over.
	items := make(map[string]Peer)
	_ = s.peers.Range(func(key string, value Peer) (bool, error) {
		items[key] = value
		return false, nil
	})

	for id := range items {
		// Prepare gRPC request to get uplink traffic stats.
		in := &statscommand.GetStatsRequest{
			Reset_: false,
			Name:   fmt.Sprintf("user>>>%s>>>traffic>>>uplink", id),
		}

		// Send the request to get uplink traffic stats.
		res, err := client.GetStats(ctx, in)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to get uplink stats: %w", err)
		}

		// Extract uplink traffic stats or use an empty stat if not found.
		txBytes := &statscommand.Stat{}
		if res != nil && res.GetStat() != nil {
			txBytes = res.GetStat()
		}

		// Prepare gRPC request to get downlink traffic stats.
		in = &statscommand.GetStatsRequest{
			Reset_: false,
			Name:   fmt.Sprintf("user>>>%s>>>traffic>>>downlink", id),
		}

		// Send the request to get downlink traffic stats.
		res, err = client.GetStats(ctx, in)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to get downlink stats: %w", err)
		}

		// Extract downlink traffic stats or use an empty stat if not found.
		rxBytes := &statscommand.Stat{}
		if res != nil && res.GetStat() != nil {
			rxBytes = res.GetStat()
		}

		s.peers.Update(id, func(p Peer, ok bool) Peer {
			if !ok {
				return p
			}

			p.Current.RxBytes = rxBytes.GetValue()
			p.Current.TxBytes = txBytes.GetValue()
			p.Current.Duration = time.Since(p.Timestamp)

			return p
		})
	}

	return nil
}
