package xray

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	procutils "github.com/shirou/gopsutil/v4/process"
	proxymancommand "github.com/xtls/xray-core/app/proxyman/command"
	statscommand "github.com/xtls/xray-core/app/stats/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sentinel-official/sentinel-go-sdk/libs/crypto"
	"github.com/sentinel-official/sentinel-go-sdk/libs/safe"
	"github.com/sentinel-official/sentinel-go-sdk/process"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Server implements types.ServerService interface.
var _ types.ServerService = (*Server)(nil)

// Server represents the Xray server instance.
type Server struct {
	*process.Manager // Embedded process manager for handling lifecycle.

	cfg     *ServerConfig // Configuration settings for the service.
	homeDir string        // Home directory of the service.

	cmd      *exec.Cmd               // Command to run the Xray server.
	conn     *safe.GRPCConn          // gRPC client connection to the service.
	metadata []*ServerMetadata       // Metadata containing server-specific details.
	peers    *safe.Map[string, Peer] // Thread-safe map to manage peers connected to the server.
	proxies  map[string]proxy        // Proxy protocols used by the server, keyed by inbound tag.
}

// proxy holds the per-inbound parameters needed to add a user over gRPC.
type proxy struct {
	Protocol ProxyProtocol // Protocol is the inbound proxy protocol.
	Flow     Flow          // Flow is the inbound VLESS flow control setting.
}

// NewServer creates a new Server instance.
func NewServer(name, appDir string, cfg *ServerConfig) *Server {
	return &Server{
		Manager: process.NewManager(name),
		cfg:     cfg,
		conn:    &safe.GRPCConn{},
		homeDir: filepath.Join(appDir, "xray"),
		peers:   safe.NewMap[string, Peer](),
		proxies: make(map[string]proxy),
	}
}

// Type returns the service type of the server.
func (s *Server) Type() types.ServiceType {
	return types.ServiceTypeXray
}

// Metadata returns the service metadata of the server.
func (s *Server) Metadata() any { return s.metadata }

// IsRunning checks if the Xray server process is running.
func (s *Server) IsRunning() (bool, error) {
	// Read PID from file.
	pid, err := s.readPID()
	if err != nil {
		return false, fmt.Errorf("reading PID: %w", err)
	}

	if pid == 0 {
		return false, nil
	}

	// Retrieve process with the given PID.
	proc, err := procutils.NewProcess(pid)
	if err != nil {
		if utils.ErrorIs(err, procutils.ErrorProcessNotRunning) {
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

	if name != xray {
		return false, nil
	}

	return true, nil
}

// Init sets up service configuration, creating directories and writing defaults unless config exists.
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

// Setup prepares the Xray server service for operation.
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

		s.cfg.TLSCertFile = filepath.Join(s.homeDir, "tls.crt")
		s.cfg.TLSKeyFile = filepath.Join(s.homeDir, "tls.key")

		// Generate the Reality keypair and shortIds for Reality inbounds.
		for _, inbound := range s.cfg.Inbounds {
			if inbound.GetTransportSecurity() != TransportSecurityReality {
				continue
			}

			if err := s.setupReality(inbound); err != nil {
				return fmt.Errorf("setting up reality: %w", err)
			}
		}

		// Generate the server-level key for Shadowsocks 2022 inbounds.
		for _, inbound := range s.cfg.Inbounds {
			if inbound.GetProxyProtocol() != ProxyProtocolShadowsocks2022 {
				continue
			}

			if err := s.setupShadowsocks(inbound); err != nil {
				return fmt.Errorf("setting up shadowsocks: %w", err)
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

		// Initialize PKI and issue a tls certificate
		pki := crypto.NewPKI(s.homeDir)
		if err := pki.Init(); err != nil {
			return fmt.Errorf("initializing PKI: %w", err)
		}

		if _, _, err := pki.Issue("tls"); err != nil {
			return fmt.Errorf("issuing TLS certificate and key: %w", err)
		}

		// Set the server metadata.
		for _, inbound := range s.cfg.Inbounds {
			metadata := &ServerMetadata{
				Port:              inbound.OutPort(),
				ProxyProtocol:     inbound.GetProxyProtocol(),
				TransportProtocol: inbound.GetTransportProtocol(),
				TransportSecurity: inbound.GetTransportSecurity(),
				Flow:              inbound.GetFlow(),
			}

			if inbound.GetProxyProtocol() == ProxyProtocolShadowsocks2022 {
				metadata.Method = inbound.GetMethod()
				metadata.Key = inbound.Key
			}

			if inbound.GetTransportSecurity() == TransportSecurityReality {
				metadata.RealityServerName = inbound.Reality.ServerNames[0]
				metadata.RealityShortId = inbound.Reality.ShortIds[0]
				metadata.RealityPublicKey = inbound.Reality.PublicKey
				metadata.RealityFingerprint = inbound.Reality.Fingerprint
			}

			s.metadata = append(s.metadata, metadata)
			s.proxies[inbound.Tag()] = proxy{
				Protocol: inbound.GetProxyProtocol(),
				Flow:     inbound.GetFlow(),
			}
		}

		return nil
	})
}

// Start starts the Xray server service.
func (s *Server) Start(parent context.Context) (context.Context, error) {
	return s.Manager.Start(parent, func(ctx context.Context) error { //nolint:wrapcheck
		// Constructs the command to start the Xray server.
		cfgFile := s.serviceConfigFile()
		s.cmd = exec.CommandContext(
			ctx,
			s.execFile(xray),
			"run", "--config", cfgFile,
		)

		// Starts the Xray server process.
		if err := s.cmd.Start(); err != nil {
			return fmt.Errorf("starting command: %w", err)
		}

		// Write PID to file.
		if err := s.writePID(s.cmd.Process.Pid); err != nil {
			return fmt.Errorf("writing PID: %w", err)
		}

		target := "127.0.0.1:2323"
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}

		// Establish a safe, concurrency-managed gRPC connection to the target.
		if err := s.conn.Dial(target, opts...); err != nil {
			return fmt.Errorf("gRPC dialing target %q: %w", target, err)
		}

		// Wait for the Xray process to finish in a separate goroutine.
		s.Go(ctx, func() (err error) {
			if err = s.cmd.Wait(); err == nil {
				err = errors.New("exited unexpectedly")
			}

			return fmt.Errorf("waiting command: %w", err)
		})

		// Start background goroutine for periodic peer statistics updates.
		s.Go(ctx, func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
					// Check if server is up before syncing peers.
					ok, err := s.IsRunning()
					if err != nil {
						return fmt.Errorf("checking service status: %w", err)
					}

					if !ok {
						continue
					}

					// Sync peer statistics from Xray.
					if err := s.syncPeers(ctx); err != nil {
						return fmt.Errorf("syncing peer statistics: %w", err)
					}
				}
			}
		})

		return nil
	})
}

// Stop stops the Xray server service.
func (s *Server) Stop() error {
	return s.Manager.Stop(func() error { //nolint:wrapcheck
		// Close gRPC client connection.
		if err := s.conn.Close(); err != nil {
			return fmt.Errorf("closing gRPC client connection: %w", err)
		}

		// Read PID from file.
		pid, err := s.readPID()
		if err != nil {
			return fmt.Errorf("reading PID: %w", err)
		}

		if pid == 0 {
			return nil
		}

		// Retrieve process with the given PID.
		proc, err := procutils.NewProcess(pid)
		if err != nil {
			if utils.ErrorIs(err, procutils.ErrorProcessNotRunning) {
				return nil
			}

			return fmt.Errorf("getting process for PID %d: %w", pid, err)
		}

		// Terminate the process.
		if err := proc.Terminate(); err != nil {
			return fmt.Errorf("terminating process: %w", err)
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
			return fmt.Errorf("removing config file %q: %w", cfgFile, err)
		}

		// Remove PID file.
		pidFile := s.pidFile()
		if err := utils.RemoveFile(pidFile); err != nil {
			return fmt.Errorf("removing PID file %q: %w", pidFile, err)
		}

		return nil
	})
}

// AddPeer adds a new peer to the Xray server.
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

	conn, release := s.conn.Acquire()
	if conn == nil {
		return "", nil, errors.New("acquiring connection: nil conn")
	}

	defer release()

	client := proxymancommand.NewHandlerServiceClient(conn)

	for tag, p := range s.proxies {
		// Prepare gRPC request to add a new user to the handler.
		in := &proxymancommand.AlterInboundRequest{
			Tag: tag,
			Operation: serial.ToTypedMessage(
				&proxymancommand.AddUserOperation{
					User: &protocol.User{
						Email:   id,
						Account: p.Protocol.Account(r.UUID, p.Flow),
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

// HasPeer checks if a peer exists in the Xray server's peer list.
func (s *Server) HasPeer(_ context.Context, id string) (bool, error) {
	return s.peers.Exists(id), nil
}

// RemovePeer removes a peer from the Xray server.
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

// PeersLen returns the number of peers connected to the Xray server.
func (s *Server) PeersLen() int {
	return s.peers.Len()
}

// PeerStatistics retrieves statistics for each peer connected to the Xray server.
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
func (s *Server) pidFile() string           { return filepath.Join(s.homeDir, "server.pid") }
func (s *Server) serviceConfigFile() string { return filepath.Join(s.homeDir, "server.json") }

// setupReality fills in the Reality defaults and generates the keypair and shortIds.
func (s *Server) setupReality(inbound *InboundServerConfig) error {
	if inbound.Reality == nil {
		inbound.Reality = &Reality{}
	}

	reality := inbound.Reality

	// Apply defaults for the operator-configurable fields.
	if reality.Dest == "" {
		reality.Dest = DefaultRealityDest
	}

	if len(reality.ServerNames) == 0 {
		host, _, err := net.SplitHostPort(reality.Dest)
		if err != nil {
			return fmt.Errorf("splitting dest %q: %w", reality.Dest, err)
		}

		reality.ServerNames = []string{host}
	}

	if reality.Fingerprint == "" {
		reality.Fingerprint = DefaultRealityFingerprint
	}

	// Generate a shortId when none is configured.
	if len(reality.ShortIds) == 0 {
		shortID, err := newShortID()
		if err != nil {
			return fmt.Errorf("generating short_id: %w", err)
		}

		reality.ShortIds = []string{shortID}
	}

	// Generate the X25519 keypair.
	if err := reality.generateKeys(); err != nil {
		return fmt.Errorf("generating keys: %w", err)
	}

	return nil
}

// setupShadowsocks fills in the Shadowsocks 2022 method default and generates the keys.
func (s *Server) setupShadowsocks(inbound *InboundServerConfig) error {
	// Apply the method default.
	inbound.Method = inbound.GetMethod()

	// Generate the server-level key (iPSK).
	key, err := newKey()
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	inbound.Key = key

	// Generate a distinct throwaway key for the inert seed user. The seed user only
	// exists to force the multi-user inbound; its key must differ from the iPSK, which
	// is client-distributed via metadata, so knowing the iPSK alone never grants access.
	seedKey, err := newKey()
	if err != nil {
		return fmt.Errorf("generating seed key: %w", err)
	}

	inbound.SeedKey = seedKey

	return nil
}

// readPID reads the PID from the server's PID file.
func (s *Server) readPID() (int32, error) {
	// Get the full path to the PID file
	pidFile := filepath.Clean(s.pidFile())

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

// writePID writes the given PID to the server's PID file.
func (s *Server) writePID(pid int) error {
	// Convert PID to byte slice.
	data := []byte(strconv.Itoa(pid))

	// Write PID to file with appropriate permissions.
	pidFile := s.pidFile()
	if err := os.WriteFile(pidFile, data, 0600); err != nil {
		return fmt.Errorf("writing PID file %q: %w", pidFile, err)
	}

	return nil
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
