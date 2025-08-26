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

	"github.com/sentinel-official/sentinel-go-sdk/libs/crypto"
	"github.com/sentinel-official/sentinel-go-sdk/libs/safe"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Server implements types.ServerService interface.
var _ types.ServerService = (*Server)(nil)

// Server represents the V2Ray server instance.
type Server struct {
	cmd      *exec.Cmd                // Command to run the V2Ray server.
	conn     *safe.GRPCConn           // gRPC client connection to the service.
	homeDir  string                   // Home directory of the V2Ray server.
	metadata []*ServerMetadata        // Metadata for the server's inbound connections.
	name     string                   // Name of the server instance.
	peers    *safe.Map[string, Peer]  // Peer manager for handling peer information.
	proxies  map[string]ProxyProtocol // Proxy protocols used by the server.

	cancel context.CancelFunc // Context cancel function to stop background tasks.
	ctx    context.Context    // Context for managing the server lifecycle.
	eg     *errgroup.Group    // Error group for managing background goroutines.
}

// NewServer creates a new Server instance.
func NewServer(appDir string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Server{
		conn:    &safe.GRPCConn{},
		homeDir: filepath.Join(appDir, "v2ray"),
		name:    "server",
		peers:   safe.NewMap[string, Peer](),
		proxies: make(map[string]ProxyProtocol),
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

// pidFilePath returns the file path of the server's PID file.
func (s *Server) pidFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.pid", s.name))
}

// serviceConfigFilePath returns the full path to the service-specific configuration file.
func (s *Server) serviceConfigFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.json", s.name))
}

// readPIDFromFile reads the PID from the server's PID file.
func (s *Server) readPIDFromFile() (int32, error) {
	// Get the full path to the PID file
	pidFile := s.pidFilePath()

	// Check if the PID file exists
	exists, err := utils.IsFileExists(pidFile)
	if err != nil {
		return 0, fmt.Errorf("checking if PID file %q exists: %w", pidFile, err)
	}
	if !exists {
		return 0, nil
	}

	// Read PID from the PID file.
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, fmt.Errorf("reading PID file %q: %w", pidFile, err)
	}

	// Convert PID data to integer.
	pid, err := strconv.ParseInt(string(data), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parsing PID: %w", err)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID %d", pid)
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
		return fmt.Errorf("writing PID file %q: %w", pidFile, err)
	}

	return nil
}

// Type returns the service type of the server.
func (s *Server) Type() types.ServiceType {
	return types.ServiceTypeV2Ray
}

// Init sets up service configuration, creating directories and writing defaults unless config exists.
func (s *Server) Init(req interface{}, force bool) error {
	// Set default server configuration
	cfg := DefaultServerConfig()

	// If a request is provided, attempt to cast it to a ServerConfig type
	if req != nil {
		if v, ok := req.(*ServerConfig); ok {
			cfg = v
		} else {
			return fmt.Errorf("invalid request type %T", req)
		}
	}

	// Create the home directory if it doesn't exist
	if err := os.MkdirAll(s.homeDir, 0755); err != nil {
		return fmt.Errorf("creating home directory %q: %w", s.homeDir, err)
	}

	// Construct the full path to the config file
	cfgFile := s.appConfigFilePath()

	// Check if the config file exists at the specified path
	exists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
	}

	// Write default config only if file doesn't exist or force flag is enabled
	if !exists || force {
		if err := cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("writing config file %q: %w", cfgFile, err)
		}
	}

	return nil
}

// IsUp checks if the V2Ray server process is running.
func (s *Server) IsUp() (bool, error) {
	// Read PID from file.
	pid, err := s.readPIDFromFile()
	if err != nil {
		return false, fmt.Errorf("reading PID from file: %w", err)
	}
	if pid == 0 {
		return false, nil
	}

	// Retrieve process with the given PID.
	proc, err := process.NewProcess(pid)
	if err != nil {
		if utils.ErrorIs(err, process.ErrorProcessNotRunning) {
			return false, nil
		}

		return false, fmt.Errorf("getting process for PID %d: %w", pid, err)
	}

	// Check if the process is running.
	ok, err := proc.IsRunning()
	if err != nil {
		return false, fmt.Errorf("checking process status: %w", err)
	}
	if !ok {
		return false, nil
	}

	// Retrieve the name of the process.
	name, err := proc.Name()
	if err != nil {
		return false, fmt.Errorf("getting process name: %w", err)
	}

	// Check if the process name matches constant v2ray.
	if name != v2ray {
		return false, nil
	}

	return true, nil
}

// PreUp writes the configuration to the config file before starting the server process.
func (s *Server) PreUp() error {
	// Construct the full path to the config file
	cfgFile := s.appConfigFilePath()

	// Check if the config file exists at the specified path
	exists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
	}

	// Initialize viper instance
	v := viper.New()

	// If the config file exists, proceed to read its contents
	if exists {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("reading config file %q: %w", cfgFile, err)
		}
	}

	// Set default server configuration
	cfg := DefaultServerConfig()

	// Unmarshal configuration into the config object
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshaling config file %q: %w", cfgFile, err)
	}

	cfg.TLSCertFile = filepath.Join(s.homeDir, "tls.crt")
	cfg.TLSKeyFile = filepath.Join(s.homeDir, "tls.key")

	// Validate the unmarshalled config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	// Initialize PKI and issue a tls certificate
	pki := crypto.NewPKI(s.homeDir)
	if err := pki.Init(); err != nil {
		return fmt.Errorf("initializing PKI: %w", err)
	}
	if _, _, err := pki.Issue("tls"); err != nil {
		return fmt.Errorf("issuing TLS certificate and key: %w", err)
	}

	for _, inbound := range cfg.Inbounds {
		metadata := &ServerMetadata{
			Port:              inbound.OutPort(),
			ProxyProtocol:     inbound.GetProxyProtocol(),
			TransportProtocol: inbound.GetTransportProtocol(),
			TransportSecurity: inbound.GetTransportSecurity(),
		}

		s.metadata = append(s.metadata, metadata)
		s.proxies[inbound.Tag()] = inbound.GetProxyProtocol()
	}

	// Write configuration to file.
	cfgFile = s.serviceConfigFilePath()
	if err := cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("writing config file %q: %w", cfgFile, err)
	}

	return nil
}

// Up starts the V2Ray server process.
func (s *Server) Up() error {
	// Constructs the command to start the V2Ray server.
	cfgFile := s.serviceConfigFilePath()
	s.cmd = exec.CommandContext(
		s.ctx,
		s.execFile(v2ray),
		strings.Fields(fmt.Sprintf("run --config %s", cfgFile))...,
	)

	// Starts the V2Ray server process.
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	// Wait for the V2Ray process to finish in a separate goroutine.
	s.eg.Go(func() (err error) {
		if err = s.cmd.Wait(); err == nil {
			err = errors.New("command exited unexpectedly")
		}

		return fmt.Errorf("waiting command: %w", err)
	})

	return nil
}

// PostUp performs operations after the server process is started.
func (s *Server) PostUp() (err error) {
	// Write PID to file.
	if err := s.writePIDToFile(s.cmd.Process.Pid); err != nil {
		return fmt.Errorf("writing PID to file: %w", err)
	}

	target := "127.0.0.1:2323"
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Establish a safe, concurrency-managed gRPC connection to the target.
	if err := s.conn.Dial(target, opts...); err != nil {
		return fmt.Errorf("gRPC dialing target %q: %w", target, err)
	}

	// Start background goroutine for periodic peer statistics updates.
	s.eg.Go(func() error {
		// Blocking loop that runs until stop signal is received
		for {
			select {
			case <-s.ctx.Done():
				return s.ctx.Err()
			case <-time.After(time.Second):
				// Check if server is up before syncing peers.
				ok, err := s.IsUp()
				if err != nil {
					return fmt.Errorf("checking serivce status: %w", err)
				}
				if !ok {
					continue
				}

				// Sync peer statistics from WireGuard.
				if err := s.syncPeers(s.ctx); err != nil {
					return fmt.Errorf("syncing peer statistics: %w", err)
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

	// Close gRPC client connection.
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("closing gRPC client connection: %w", err)
	}

	return nil
}

// Down terminates the V2Ray server process.
func (s *Server) Down() error {
	// Read PID from file.
	pid, err := s.readPIDFromFile()
	if err != nil {
		return fmt.Errorf("reading PID from file: %w", err)
	}
	if pid == 0 {
		return nil
	}

	// Retrieve process with the given PID.
	proc, err := process.NewProcess(pid)
	if err != nil {
		if utils.ErrorIs(err, process.ErrorProcessNotRunning) {
			return nil
		}

		return fmt.Errorf("getting process for PID %d: %w", pid, err)
	}

	// Terminate the process.
	if err := proc.Terminate(); err != nil {
		return fmt.Errorf("terminating process: %w", err)
	}

	return nil
}

// PostDown performs cleanup operations after the server process is terminated.
func (s *Server) PostDown() error {
	// Removes configuration file.
	cfgFile := s.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("removing config file %q: %w", cfgFile, err)
	}

	// Remove PID file.
	pidFile := s.pidFilePath()
	if err := utils.RemoveFile(pidFile); err != nil {
		return fmt.Errorf("removing PID file %q: %w", pidFile, err)
	}

	return nil
}

// AddPeer adds a new peer to the V2Ray server.
func (s *Server) AddPeer(ctx context.Context, req interface{}) (string, interface{}, error) {
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

	conn, release := s.conn.Acquire()
	if conn == nil {
		return "", nil, errors.New("acquiring connection: nil conn")
	}

	defer release()

	client := proxymancommand.NewHandlerServiceClient(conn)
	for tag, proxy := range s.proxies {
		// Prepare gRPC request to add a new user to the handler.
		in := &proxymancommand.AlterInboundRequest{
			Tag: tag,
			Operation: serial.ToTypedMessage(
				&proxymancommand.AddUserOperation{
					User: &protocol.User{
						Email:   id,
						Account: proxy.Account(r.UUID),
					},
				},
			),
		}

		// Send the request to add a user to the handler.
		if _, err := client.AlterInbound(ctx, in); err != nil {
			return "", nil, fmt.Errorf("altering peer %q inbound: %w", id, err)
		}
	}

	// Save the peer details in the local peers map.
	now := time.Now()
	s.peers.Set(id, Peer{
		ID:       id,
		Current:  types.NewPeerStatistics(now),
		Previous: types.NewPeerStatistics(now),
	})

	// Return nil for success (no additional data to return in response).
	return id, &AddPeerResponse{
		Metadata: s.metadata,
	}, nil
}

// HasPeer checks if a peer exists in the V2Ray server's peer list.
func (s *Server) HasPeer(_ context.Context, id string) (bool, error) {
	return s.peers.Exists(id), nil
}

// RemovePeer removes a peer from the V2Ray server.
func (s *Server) RemovePeer(ctx context.Context, id string) error {
	conn, release := s.conn.Acquire()
	if conn == nil {
		return errors.New("acquiring connection: nil conn")
	}

	defer release()

	client := proxymancommand.NewHandlerServiceClient(conn)
	for tag := range s.proxies {
		// Prepare gRPC request to remove a user from the handler.
		in := &proxymancommand.AlterInboundRequest{
			Tag: tag,
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
				return fmt.Errorf("altering peer %q inbound: %w", id, err)
			}
		}
	}

	// Remove the peer information from the local collection.
	s.peers.Delete(id, nil)
	return nil
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

// syncPeers retrieves the latest peer transfer statistics from the stats service
// and updates the in-memory peer data accordingly.
func (s *Server) syncPeers(ctx context.Context) error {
	// Prepare the response
	resp := &statscommand.QueryStatsResponse{}

	// Perform the gRPC call to fetch traffic stats
	fn := func() (err error) {
		conn, release := s.conn.Acquire()
		if conn == nil {
			return errors.New("acquiring connection: nil conn")
		}

		defer release()

		client := statscommand.NewStatsServiceClient(conn)

		// Send the request to get traffic stats
		resp, err = client.QueryStats(ctx, &statscommand.QueryStatsRequest{})
		if err != nil {
			return fmt.Errorf("querying peer stats: %w", err)
		}

		return nil
	}

	// Execute stats query
	if err := fn(); err != nil {
		return err
	}

	// Temporary map to collect per-user traffic stats
	stats := make(map[string]*types.PeerStatistics)

	// Iterate over every stat entry
	for _, stat := range resp.GetStat() {
		name := stat.GetName()

		// Split the name into 4 parts
		parts := strings.SplitN(name, ">>>", 4)
		if len(parts) != 4 || parts[0] != "user" || parts[2] != "traffic" {
			continue
		}

		id := parts[1]

		// Skip if the peer does not exist
		if !s.peers.Exists(id) {
			continue
		}

		// Get or create statistics entry
		pt, exists := stats[id]
		if !exists {
			pt = &types.PeerStatistics{}
			stats[id] = pt
		}

		// Assign Rx/Tx values based on direction
		switch parts[3] {
		case "uplink":
			pt.RxBytes = stat.GetValue()
		case "downlink":
			pt.TxBytes = stat.GetValue()
		default:
			continue
		}
	}

	createdAt := time.Time{}
	now := time.Now()

	// Apply collected stats to the thread-safe map
	for id, stat := range stats {
		s.peers.Update(id, func(v Peer, ok bool) (Peer, bool) {
			if !ok {
				return v, false
			}

			if createdAt.After(v.Current.CreatedAt) {
				v.Previous.RxBytes += v.Current.RxBytes
				v.Previous.TxBytes += v.Current.TxBytes
				v.Previous.UpdatedAt = now

				v.Current = types.NewPeerStatistics(createdAt)
			}

			if v.Current.RxBytes > 0 && stat.RxBytes == v.Current.RxBytes {
				return v, false
			}

			v.Current.RxBytes = stat.RxBytes
			v.Current.TxBytes = stat.TxBytes
			v.Current.UpdatedAt = now

			return v, true
		})
	}

	return nil
}
