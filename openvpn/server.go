package openvpn

import (
	"bufio"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
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
	cfg      *ServerConfig           // Configuration settings for the OpenVPN server.
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
func NewServer(appDir string, cfg *ServerConfig) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Server{
		cfg:     cfg,
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

// pidFilePath returns the full path to the PID file for OpenVPN process.
func (s *Server) pidFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.pid", s.name))
}

// serviceConfigFilePath returns the full path to the service-specific configuration file.
func (s *Server) serviceConfigFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.conf", s.name))
}

// readPIDFromFile reads the PID of the running OpenVPN process from a file.
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

// writePIDToFile saves the given PID to a file for later reference.
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

// mgmtConn creates a TCP connection to the OpenVPN management interface.
func (s *Server) mgmtConn() (net.Conn, error) {
	target := "127.0.0.1:2323"
	timeout := 5 * time.Second

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return nil, fmt.Errorf("creating TCP connection to target %q: %w", target, err)
	}

	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("setting connection deadline %q: %w", deadline, err)
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
		return fmt.Errorf("creating home directory %q: %w", s.homeDir, err)
	}

	// Construct the full path to the config file
	cfgFile := s.appConfigFilePath()

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

// IsUp checks whether the OpenVPN server is running by verifying its PID and process name.
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

	// Check if the process name matches constant openVPN.
	if name != openVPN {
		return false, nil
	}

	return true, nil
}

// PreUp prepares the server before it is started by initializing PKI and generating config files.
func (s *Server) PreUp() error {
	// Construct the full path to the config file
	cfgFile := s.appConfigFilePath()

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

	s.cfg.PKIDir = filepath.Join(s.homeDir, "pki")
	s.cfg.StatusFile = filepath.Join(s.homeDir, "server.log")

	// Validate the config
	if err := s.cfg.Validate(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	// Initialize PKI and issue a server certificate
	s.pki = crypto.NewPKI(s.cfg.PKIDir)
	if err := s.pki.Init(); err != nil {
		return fmt.Errorf("initializing PKI: %w", err)
	}
	if _, _, err := s.pki.Issue("server"); err != nil {
		return fmt.Errorf("issuing server certificate and key: %w", err)
	}

	// Generate a TLS static key
	tlsBuf := make([]byte, 256)
	if _, err := rand.Read(tlsBuf); err != nil {
		return fmt.Errorf("generating TLS static key: %w", err)
	}

	tlsFile := filepath.Join(s.cfg.PKIDir, "tls.key")
	if err := pem.WriteFile(tlsFile, pem.FormatHex, pem.BlockTypeOpenVPNStaticKeyV1, tlsBuf); err != nil {
		return fmt.Errorf("writing TLS static key file %q: %w", tlsFile, err)
	}

	// Save server metadata for later use
	s.metadata = []*ServerMetadata{
		{
			Port:     s.cfg.OutPort(),
			Protocol: s.cfg.Protocol,
			CA:       s.pki.Certificate.Raw,
			TLS:      tlsBuf,
		},
	}

	// Write configuration to file.
	cfgFile = s.serviceConfigFilePath()
	if err := s.cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("writing service config file %q: %w", cfgFile, err)
	}

	return nil
}

// Up launches the OpenVPN server using the generated config file.
func (s *Server) Up() error {
	// Constructs the command to start the OpenVPN server.
	cfgFile := s.serviceConfigFilePath()
	s.cmd = exec.CommandContext(
		s.ctx,
		s.execFile(openVPN),
		strings.Fields(fmt.Sprintf("--config %s", cfgFile))...,
	)

	// Starts the OpenVPN server process.
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	// Waits for the process to complete in a separate goroutine.
	s.eg.Go(func() (err error) {
		if err = s.cmd.Wait(); err == nil {
			err = errors.New("command exited unexpectedly")
		}

		return fmt.Errorf("waiting command: %w", err)
	})

	return nil
}

// PostUp stores the PID of the running process and waits for process completion.
func (s *Server) PostUp() error {
	// Write PID to file.
	if err := s.writePIDToFile(s.cmd.Process.Pid); err != nil {
		return fmt.Errorf("writing PID to file: %w", err)
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
					return fmt.Errorf("checking service status: %w", err)
				}
				if !ok {
					continue
				}

				// Sync peer statistics from OpenVPN.
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
		return fmt.Errorf("reading PID from file: %w", err)
	}
	if pid == 0 {
		return nil
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		if utils.ErrorIs(err, process.ErrorProcessNotRunning) {
			return nil
		}

		return fmt.Errorf("getting process: %w", err)
	}

	if err := proc.Terminate(); err != nil {
		return fmt.Errorf("terminating process: %w", err)
	}

	return nil
}

// PostDown removes PID file after the process has stopped.
func (s *Server) PostDown() error {
	// Removes configuration file.
	cfgFile := s.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("removing config file %q: %w", cfgFile, err)
	}

	pidFile := s.pidFilePath()
	if err := utils.RemoveFile(pidFile); err != nil {
		return fmt.Errorf("removing PID file %q: %w", pidFile, err)
	}

	return nil
}

// AddPeer creates and registers a new VPN peer by issuing a new certificate.
func (s *Server) AddPeer(_ context.Context, req interface{}) (string, interface{}, error) {
	// Parse the request to PeerRequest type.
	r, err := parsePeerRequest(req)
	if err != nil {
		return "", nil, fmt.Errorf("parsing request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return "", nil, fmt.Errorf("validating request: %w", err)
	}

	id := r.ID()

	keyDER, certDER, err := s.pki.Issue(id)
	if err != nil {
		return "", nil, fmt.Errorf("issuing %q certificate and key: %w", id, err)
	}

	now := time.Now()
	s.peers.Set(id, Peer{
		ID:       id,
		Current:  types.NewPeerStatistics(now),
		Previous: types.NewPeerStatistics(now),
	})

	return id, &AddPeerResponse{
		Metadata: s.metadata,
		Cert:     certDER,
		Key:      keyDER,
	}, nil
}

// HasPeer checks if a peer is currently connected and tracked.
func (s *Server) HasPeer(_ context.Context, id string) (bool, error) {
	return s.peers.Exists(id), nil
}

// RemovePeer disconnects a VPN client and revokes its certificate.
func (s *Server) RemovePeer(_ context.Context, id string) error {
	conn, err := s.mgmtConn()
	if err != nil {
		return fmt.Errorf("getting management connection: %w", err)
	}

	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)

	// Wait for management interface banner
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading line: %w", err)
		}
		if strings.Contains(line, "OpenVPN Management Interface") {
			break
		}
	}

	// Issue kill command for the client
	if _, err := fmt.Fprintf(conn, "kill %s\n", id); err != nil {
		return fmt.Errorf("writing peer %q kill command: %w", id, err)
	}

	// Await confirmation
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading line: %w", err)
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SUCCESS") {
			break
		}
		if strings.HasPrefix(line, "ERROR") {
			return fmt.Errorf("killing peer %q client: %s", id, line)
		}
	}

	// Revoke the certificate
	if err := s.pki.Revoke(id); err != nil {
		return fmt.Errorf("revoking %q certificate: %w", id, err)
	}

	s.peers.Delete(id, nil)
	return nil
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

func (s *Server) syncPeers(_ context.Context) error {
	conn, err := s.mgmtConn()
	if err != nil {
		return fmt.Errorf("getting management connection: %w", err)
	}

	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)

	// Wait for the management interface welcome message
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading line: %w", err)
		}
		if strings.Contains(line, "OpenVPN Management Interface") {
			break
		}
	}

	// Request status output
	if _, err := fmt.Fprintf(conn, "status 2\n"); err != nil {
		return fmt.Errorf("writing status command: %w", err)
	}

	// Parse client statistics
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading line: %w", err)
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
			return fmt.Errorf("parsing peer %q uplink bytes %q: %w", id, fields[5], err)
		}

		txBytes, err := strconv.ParseInt(fields[6], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing peer %q downlink bytes %q: %w", id, fields[6], err)
		}

		createdAt, err := time.Parse(time.DateTime, fields[7])
		if err != nil {
			return fmt.Errorf("parsing peer %q created at %q: %w", id, fields[7], err)
		}

		now := time.Now()

		// Update peer statistics in thread-safe map.
		s.peers.Update(id, func(v Peer, found bool) (Peer, bool) {
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
