package hysteria2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	procutils "github.com/shirou/gopsutil/v4/process"

	"github.com/sentinel-official/sentinel-go-sdk/libs/crypto"
	"github.com/sentinel-official/sentinel-go-sdk/libs/safe"
	"github.com/sentinel-official/sentinel-go-sdk/process"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Server implements types.ServerService interface.
var _ types.ServerService = (*Server)(nil)

// Server represents the Hysteria2 server instance.
type Server struct {
	*process.Manager // Embedded process manager for handling lifecycle.

	cfg     *ServerConfig // Configuration settings for the service.
	homeDir string        // Home directory of the service.

	cmd        *exec.Cmd               // Command to run the Hysteria2 server.
	httpClient *http.Client            // HTTP client for loopback API calls (auth, stats, kick).
	metadata   []*ServerMetadata       // Metadata containing server-specific details.
	peers      *safe.Map[string, Peer] // Thread-safe map to manage peers connected to the server.
}

// NewServer creates a new Server instance.
func NewServer(name, appDir string, cfg *ServerConfig) *Server {
	return &Server{
		Manager:    process.NewManager(name),
		cfg:        cfg,
		homeDir:    filepath.Join(appDir, hysteria2),
		httpClient: &http.Client{Timeout: 5 * time.Second},
		peers:      safe.NewMap[string, Peer](),
	}
}

// Type returns the service type of the server.
func (s *Server) Type() types.ServiceType {
	return types.ServiceTypeHysteria2
}

// Metadata returns the service metadata of the server.
func (s *Server) Metadata() any { return s.metadata }

// IsRunning checks if the Hysteria2 server process is running.
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

	if name != hysteria2 {
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

// Setup prepares the Hysteria2 server service for operation.
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

		// Validate the config
		if err := s.cfg.Validate(); err != nil {
			return fmt.Errorf("validating config: %w", err)
		}

		// Write configuration to file.
		cfgFile = s.serviceConfigFile()
		if err := s.cfg.WriteServiceConfig(cfgFile); err != nil {
			return fmt.Errorf("writing service config file %q: %w", cfgFile, err)
		}

		// Initialize PKI and issue a TLS certificate.
		pki := crypto.NewPKI(s.homeDir)
		if err := pki.Init(); err != nil {
			return fmt.Errorf("initializing PKI: %w", err)
		}

		_, certDER, err := pki.Issue("tls")
		if err != nil {
			return fmt.Errorf("issuing TLS certificate and key: %w", err)
		}

		// Compute SHA-256 fingerprint of the TLS certificate.
		tlsPin, err := certPin(certDER)
		if err != nil {
			return fmt.Errorf("computing TLS pin: %w", err)
		}

		// Set the server metadata.
		s.metadata = []*ServerMetadata{
			{
				Port:         s.cfg.Port,
				TLSPin:       tlsPin,
				ObfsPassword: s.cfg.ObfsPassword,
			},
		}

		return nil
	})
}

// Start starts the Hysteria2 server service.
func (s *Server) Start(parent context.Context) (context.Context, error) {
	return s.Manager.Start(parent, func(ctx context.Context) error { //nolint:wrapcheck
		// Bind the loopback auth listener before starting the process so that
		// Hysteria2 cannot race ahead and call the auth backend before it is up.
		authAddr := fmt.Sprintf("127.0.0.1:%d", s.cfg.AuthPort)

		authListener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", authAddr)
		if err != nil {
			return fmt.Errorf("listening on auth address %q: %w", authAddr, err)
		}

		// Constructs the command to start the Hysteria2 server.
		cfgFile := s.serviceConfigFile()
		s.cmd = exec.CommandContext(
			ctx,
			s.execFile(hysteria2),
			"server", "-c", cfgFile,
		)
		s.cmd.Stderr = os.Stderr

		// Starts the Hysteria2 server process.
		if err := s.cmd.Start(); err != nil {
			_ = authListener.Close()

			return fmt.Errorf("starting command: %w", err)
		}

		// Write PID to file.
		if err := s.writePID(s.cmd.Process.Pid); err != nil {
			_ = authListener.Close()

			return fmt.Errorf("writing PID: %w", err)
		}

		// Wait for the Hysteria2 process to finish in a separate goroutine.
		s.Go(ctx, func() (err error) {
			if err = s.cmd.Wait(); err == nil {
				err = errors.New("exited unexpectedly")
			}

			return fmt.Errorf("waiting command: %w", err)
		})

		// Start the loopback HTTP auth backend on the already-bound listener.
		authSrv := &http.Server{
			Handler:           newAuthHandler(s.peers),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      5 * time.Second,
		}

		s.Go(ctx, func() error {
			if err := authSrv.Serve(authListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("running auth server: %w", err)
			}

			return nil
		})

		// Shut down the auth server when the context is canceled.
		s.Go(ctx, func() error {
			<-ctx.Done()

			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := authSrv.Shutdown(shutCtx); err != nil { //nolint:contextcheck
				return fmt.Errorf("shutting down auth server: %w", err)
			}

			return nil
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

					// Sync peer statistics from Hysteria2.
					if err := s.syncPeers(ctx); err != nil {
						return fmt.Errorf("syncing peer statistics: %w", err)
					}
				}
			}
		})

		return nil
	})
}

// Stop stops the Hysteria2 server service.
func (s *Server) Stop() error {
	return s.Manager.Stop(func() error { //nolint:wrapcheck
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

// AddPeer adds a new peer to the Hysteria2 server.
func (s *Server) AddPeer(_ context.Context, req any) (string, any, error) {
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

	// Save the peer details in the local peers map.
	now := time.Now()
	s.peers.Set(id, Peer{
		ID:       id,
		Current:  types.NewPeerStatistics(now),
		Previous: types.NewPeerStatistics(now),
	})

	return id, &AddPeerResponse{
		Metadata: s.metadata,
	}, nil
}

// HasPeer checks if a peer exists in the Hysteria2 server's peer list.
func (s *Server) HasPeer(_ context.Context, id string) (bool, error) {
	return s.peers.Exists(id), nil
}

// RemovePeer removes a peer from the Hysteria2 server and kicks any active session.
func (s *Server) RemovePeer(ctx context.Context, id string) error {
	// Remove the peer information from the local collection.
	s.peers.Delete(id, nil)

	// POST /kick to drop any active session.
	if err := s.kickPeer(ctx, id); err != nil {
		return fmt.Errorf("kicking peer %q: %w", id, err)
	}

	return nil
}

// PeersLen returns the number of peers connected to the Hysteria2 server.
func (s *Server) PeersLen() int {
	return s.peers.Len()
}

// PeerStatistics retrieves statistics for each peer connected to the Hysteria2 server.
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
func (s *Server) serviceConfigFile() string { return filepath.Join(s.homeDir, "server.yaml") }

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

// kickPeer sends a POST /kick request to drop the active session of the given peer.
func (s *Server) kickPeer(ctx context.Context, id string) error {
	// Encode the list of IDs to kick.
	body, err := json.Marshal([]string{id})
	if err != nil {
		return fmt.Errorf("encoding kick request: %w", err)
	}

	// Build the kick URL.
	url := fmt.Sprintf("http://127.0.0.1:%d/kick", s.cfg.StatsPort)

	// Create the HTTP request.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating kick request: %w", err)
	}

	req.Header.Set("Authorization", s.cfg.StatsSecret)
	req.Header.Set("Content-Type", "application/json")

	// Send the request.
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending kick request: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	// Tolerate 404 (peer already gone); any other non-200 is an error.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("kick request returned status %d", resp.StatusCode)
	}

	return nil
}

// trafficEntry holds the per-user traffic stats returned by the Hysteria2 stats API.
type trafficEntry struct {
	Tx uint64 `json:"tx"` // Downlink bytes sent to the client.
	Rx uint64 `json:"rx"` // Uplink bytes received from the client.
}

// syncPeers retrieves the latest peer transfer statistics from Hysteria2's Traffic Stats API
// and updates the in-memory peer data accordingly.
func (s *Server) syncPeers(ctx context.Context) error {
	// Build the traffic URL.
	url := fmt.Sprintf("http://127.0.0.1:%d/traffic", s.cfg.StatsPort)

	// Create the HTTP request.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating traffic request: %w", err)
	}

	req.Header.Set("Authorization", s.cfg.StatsSecret)

	// Send the request.
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending traffic request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("traffic request returned status %d", resp.StatusCode)
	}

	// Decode the response: map of id → {tx, rx}.
	var traffic map[string]trafficEntry
	if err := json.NewDecoder(resp.Body).Decode(&traffic); err != nil {
		return fmt.Errorf("decoding traffic response: %w", err)
	}

	createdAt := time.Time{}
	now := time.Now()

	// Apply collected stats to the thread-safe map.
	for id, entry := range traffic {
		// Skip peers not in our map.
		if !s.peers.Exists(id) {
			continue
		}

		rxBytes := int64(entry.Rx)
		txBytes := int64(entry.Tx)

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

// certPin computes the SHA-256 fingerprint of a DER-encoded certificate
// and returns it as a colon-separated hex string (e.g. "ab:cd:ef:...").
func certPin(certDER []byte) (string, error) {
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return "", fmt.Errorf("parsing certificate: %w", err)
	}

	sum := sha256.Sum256(cert.Raw)

	// Encode as a hex string with colon separators (standard fingerprint format).
	parts := make([]string, sha256.Size)
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02x", b)
	}

	return strings.Join(parts, ":"), nil
}
