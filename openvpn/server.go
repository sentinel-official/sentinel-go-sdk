package openvpn

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/libs/crypto"
	"github.com/sentinel-official/sentinel-go-sdk/libs/encoding/pem"
	"github.com/sentinel-official/sentinel-go-sdk/libs/safe"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

var _ types.ServerService = (*Server)(nil)

// Server represents an OpenVPN server instance.
type Server struct {
	cmd      *exec.Cmd               // OpenVPN process command
	homeDir  string                  // Home directory where config and PID files are stored
	metadata []*ServerMetadata       // Server metadata such as port and certificates
	name     string                  // Name of the server instance.
	peers    *safe.Map[string, Peer] // Peer manager for handling peer information.
	pki      *crypto.PKI             // Public Key Infrastructure for managing certs

	cancel context.CancelFunc // Context cancel function to stop background tasks.
	ctx    context.Context    // Context for server lifecycle management.
	eg     *errgroup.Group    // Error group for managing background goroutines.
}

// NewServer creates a new Server instance.
func NewServer(appDir string) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Server{
		homeDir: filepath.Join(appDir, "openvpn"),
		name:    "server",
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

// pidFilePath returns the full path to the PID file for OpenVPN process.
func (s *Server) pidFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.pid", s.name))
}

// readPIDFromFile reads the PID of the running OpenVPN process from a file.
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

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	pid, err := strconv.ParseInt(string(data), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse pid: %w", err)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("invalid pid %d", pid)
	}

	return int32(pid), nil
}

// writePIDToFile saves the given PID to a file for later reference.
func (s *Server) writePIDToFile(pid int) error {
	data := []byte(strconv.Itoa(pid))
	if err := os.WriteFile(s.pidFilePath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// mgmtConn creates a TCP connection to the OpenVPN management interface.
func (s *Server) mgmtConn() (net.Conn, error) {
	timeout := 5 * time.Second

	conn, err := net.DialTimeout("tcp", "127.0.0.1:2323", timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to management server: %w", err)
	}

	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set connection deadline: %w", err)
	}

	return conn, nil
}

// Type returns the service type (OpenVPN).
func (s *Server) Type() types.ServiceType {
	return types.ServiceTypeOpenVPN
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

// IsUp checks whether the OpenVPN server is running by verifying its PID and process name.
func (s *Server) IsUp() (bool, error) {
	pid, err := s.readPIDFromFile()
	if err != nil {
		return false, fmt.Errorf("failed to read pid from file: %w", err)
	}
	if pid == 0 {
		return false, nil
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		if utils.ErrorIs(err, process.ErrorProcessNotRunning) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get process: %w", err)
	}

	ok, err := proc.IsRunning()
	if err != nil {
		return false, fmt.Errorf("failed to check running process: %w", err)
	}
	if !ok {
		return false, nil
	}

	name, err := proc.Name()
	if err != nil {
		return false, fmt.Errorf("failed to get process name: %w", err)
	}
	if name != openVPN {
		return false, nil
	}

	return true, nil
}

// PreUp prepares the server before it is started by initializing PKI and generating config files.
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

	cfg.PKIDir = filepath.Join(s.homeDir, "pki")
	cfg.StatusFile = filepath.Join(s.homeDir, "server.log")

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to validate config file: %w", err)
	}

	// Initialize PKI and issue a server certificate
	s.pki = crypto.NewPKI(cfg.PKIDir)
	if err := s.pki.Init(); err != nil {
		return fmt.Errorf("failed to init PKI: %w", err)
	}
	if _, _, err := s.pki.Issue("server"); err != nil {
		return fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Generate a TLS static key
	tlsBuf := make([]byte, 256)
	if _, err := rand.Read(tlsBuf); err != nil {
		return fmt.Errorf("failed to generate TLS static key: %w", err)
	}

	tlsPath := filepath.Join(cfg.PKIDir, "tls.key")
	if err := pem.WriteFile(tlsPath, pem.FormatHex, pem.BlockTypeOpenVPNStaticKeyV1, tlsBuf); err != nil {
		return fmt.Errorf("failed to write TLS static key: %w", err)
	}

	// Save server metadata for later use
	s.metadata = []*ServerMetadata{
		{
			Port:     cfg.OutPort(),
			Protocol: cfg.Protocol,
			CA:       s.pki.Certificate.Raw,
			TLS:      tlsBuf,
		},
	}

	// Write OpenVPN configuration to file
	cfgFile = s.serviceConfigFilePath()
	if err := cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("failed to write config to file: %w", err)
	}

	return nil
}

// Up launches the OpenVPN server using the generated config file.
func (s *Server) Up(ctx context.Context) error {
	// Create a context that cancels if either server or input context is done.
	ctx, _ = utils.AnyDoneContext(s.ctx, ctx)

	// Constructs the command to start the OpenVPN server.
	cfgFile := s.serviceConfigFilePath()
	s.cmd = exec.CommandContext(
		ctx,
		s.execFile(openVPN),
		strings.Fields(fmt.Sprintf("--config %s", cfgFile))...,
	)

	// Starts the OpenVPN server process.
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Waits for the process to complete in a separate goroutine.
	s.eg.Go(func() error {
		if err := s.cmd.Wait(); err != nil {
			return fmt.Errorf("failed to wait command: %w", err)
		}

		return nil
	})

	return nil
}

// PostUp stores the PID of the running process and waits for process completion.
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

// PreDown is a no-op for now but can be used for pre-shutdown tasks.
func (s *Server) PreDown() error {
	// Cancel background tasks if any.
	s.cancel()

	return nil
}

// Down gracefully stops the OpenVPN process using its PID.
func (s *Server) Down() error {
	pid, err := s.readPIDFromFile()
	if err != nil {
		return fmt.Errorf("failed to read pid from file: %w", err)
	}
	if pid == 0 {
		return nil
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		if utils.ErrorIs(err, process.ErrorProcessNotRunning) {
			return nil
		}

		return fmt.Errorf("failed to get process: %w", err)
	}

	if err := proc.Terminate(); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	return nil
}

// PostDown removes PID file after the process has stopped.
func (s *Server) PostDown() error {
	// Removes configuration file.
	cfgFile := s.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	pidFile := s.pidFilePath()
	if err := utils.RemoveFile(pidFile); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

// AddPeer creates and registers a new VPN peer by issuing a new certificate.
func (s *Server) AddPeer(_ context.Context, req interface{}) (string, interface{}, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return "", nil, fmt.Errorf("invalid request: %w", err)
	}

	id := r.ID()

	keyDER, certDER, err := s.pki.Issue(id)
	if err != nil {
		return "", nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	s.peers.Set(id, Peer{
		ID:        id,
		Current:   &types.PeerStatistics{},
		Previous:  &types.PeerStatistics{},
		Timestamp: time.Now(),
	})

	return id, &AddPeerResponse{
		Metadata: s.metadata,
		Cert:     certDER,
		Key:      keyDER,
	}, nil
}

// HasPeer checks if a peer is currently connected and tracked.
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

// RemovePeer disconnects a VPN client and revokes its certificate.
func (s *Server) RemovePeer(_ context.Context, req interface{}) (string, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return "", fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	conn, err := s.mgmtConn()
	if err != nil {
		return "", fmt.Errorf("failed to get management connection: %w", err)
	}

	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)

	// Wait for management interface banner
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read line: %w", err)
		}
		if strings.Contains(line, "OpenVPN Management Interface") {
			break
		}
	}

	// Issue kill command for the client
	id := r.ID()
	if _, err := fmt.Fprintf(conn, "kill %s\n", id); err != nil {
		return "", fmt.Errorf("failed to write command: %w", err)
	}

	// Await confirmation
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read line: %w", err)
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SUCCESS") {
			break
		}
		if strings.HasPrefix(line, "ERROR") {
			return "", fmt.Errorf("failed to kill client: %s", line)
		}
	}

	// Revoke the certificate
	if err := s.pki.Revoke(id); err != nil {
		return "", fmt.Errorf("failed to revoke certificate: %w", err)
	}

	s.peers.Delete(id)
	return id, nil
}

// PeersLen returns the number of active peers.
func (s *Server) PeersLen() int {
	return s.peers.Len()
}

// PeerStatistics queries the OpenVPN management interface and returns peer usage data.
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

func (s *Server) syncPeers(_ context.Context) error {
	conn, err := s.mgmtConn()
	if err != nil {
		return fmt.Errorf("failed to get management connection: %w", err)
	}

	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)

	// Wait for the management interface welcome message
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read line: %w", err)
		}
		if strings.Contains(line, "OpenVPN Management Interface") {
			break
		}
	}

	// Request status output
	if _, err := fmt.Fprintf(conn, "status 2\n"); err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}

	// Parse client statistics
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read line: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "END" {
			break
		}

		if !strings.HasPrefix(line, "CLIENT_LIST") {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) != 13 {
			continue
		}

		id := fields[1]
		if id == "UNDEF" {
			continue
		}

		rxBytes, err := strconv.ParseInt(fields[5], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse download bytes: %w", err)
		}

		txBytes, err := strconv.ParseInt(fields[6], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse upload bytes: %w", err)
		}

		timestamp, err := time.Parse(time.DateTime, fields[7])
		if err != nil {
			return fmt.Errorf("failed to parse timestamp: %w", err)
		}

		// Update peer statistics in thread-safe map.
		s.peers.Update(id, func(p Peer, ok bool) Peer {
			if !ok {
				return p
			}

			if timestamp.After(p.Timestamp) {
				p.Previous.Duration += p.Current.Duration
				p.Previous.RxBytes += p.Current.RxBytes
				p.Previous.TxBytes += p.Current.TxBytes

				p.Timestamp = timestamp
			}

			p.Current.Duration = time.Since(p.Timestamp)
			p.Current.RxBytes = rxBytes
			p.Current.TxBytes = txBytes

			return p
		})
	}

	return nil
}
