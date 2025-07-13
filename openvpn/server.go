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

	"github.com/sentinel-official/sentinel-go-sdk/libs/crypto"
	"github.com/sentinel-official/sentinel-go-sdk/libs/encoding/pem"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

var _ types.ServerService = (*Server)(nil)

// Server represents an OpenVPN server instance.
type Server struct {
	cmd      *exec.Cmd         // OpenVPN process command
	homeDir  string            // Home directory where config and PID files are stored
	metadata []*ServerMetadata // Server metadata such as port and certificates
	pki      *crypto.PKI       // Public Key Infrastructure for managing certs
}

// WithHomeDir sets the working directory for the server.
func (s *Server) WithHomeDir(homeDir string) *Server {
	s.homeDir = homeDir
	return s
}

// configFilePath returns the full path to the OpenVPN config file.
func (s *Server) configFilePath() string {
	return filepath.Join(s.homeDir, "openvpn.conf")
}

// pidFilePath returns the full path to the PID file for OpenVPN process.
func (s *Server) pidFilePath() string {
	return filepath.Join(s.homeDir, "openvpn.pid")
}

// readPIDFromFile reads the PID of the running OpenVPN process from a file.
func (s *Server) readPIDFromFile() (int32, error) {
	name := s.pidFilePath()
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return 0, nil
	}

	data, err := os.ReadFile(name)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	pid, err := strconv.ParseInt(string(data), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse pid: %w", err)
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

// IsUp checks whether the OpenVPN server is running by verifying its PID and process name.
func (s *Server) IsUp(ctx context.Context) (bool, error) {
	pid, err := s.readPIDFromFile()
	if err != nil {
		return false, fmt.Errorf("failed to read pid from file: %w", err)
	}
	if pid == 0 {
		return false, nil
	}

	proc, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get process: %w", err)
	}

	ok, err := proc.IsRunningWithContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check running process: %w", err)
	}
	if !ok {
		return false, nil
	}

	name, err := proc.NameWithContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get process name: %w", err)
	}
	if name != openVPN {
		return false, nil
	}

	return true, nil
}

// PreUp prepares the server before it is started by initializing PKI and generating config files.
func (s *Server) PreUp(v interface{}) error {
	cfg, ok := v.(*ServerConfig)
	if !ok {
		return fmt.Errorf("invalid parameter type %T", v)
	}

	// Initialize PKI and issue a server certificate
	s.pki = crypto.NewPKI(cfg.PKIDir)
	if err := s.pki.Init(); err != nil {
		return fmt.Errorf("failed to init PKI: %w", err)
	}
	if _, _, err := s.pki.Issue("server"); err != nil {
		return fmt.Errorf("failed to issue certificate: %w", err)
	}

	// Generate a static TLS key
	tlsBuf := make([]byte, 256)
	if _, err := rand.Read(tlsBuf); err != nil {
		return fmt.Errorf("failed to generate TLS key: %w", err)
	}

	tlsPath := filepath.Join(cfg.PKIDir, "tls.key")
	if err := pem.WriteFile(tlsPath, pem.FormatHex, pem.BlockTypeOpenVPNStaticKeyV1, tlsBuf); err != nil {
		return fmt.Errorf("failed to write TLS key: %w", err)
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
	if err := cfg.WriteToFile(s.configFilePath()); err != nil {
		return fmt.Errorf("failed to write config to file: %w", err)
	}

	return nil
}

// Up launches the OpenVPN server using the generated config file.
func (s *Server) Up(ctx context.Context) error {
	s.cmd = exec.CommandContext(
		ctx,
		s.execFile(openVPN),
		strings.Fields(fmt.Sprintf("--config %s", s.configFilePath()))...,
	)
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	return nil
}

// PostUp stores the PID of the running process and waits for process completion.
func (s *Server) PostUp() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("nil command or process")
	}

	if err := s.writePIDToFile(s.cmd.Process.Pid); err != nil {
		return fmt.Errorf("failed to write pid to file: %w", err)
	}

	if err := s.cmd.Wait(); err != nil {
		return fmt.Errorf("failed to wait for command: %w", err)
	}

	return nil
}

// PreDown is a no-op for now but can be used for pre-shutdown tasks.
func (s *Server) PreDown() error {
	return nil
}

// Down gracefully stops the OpenVPN process using its PID.
func (s *Server) Down(ctx context.Context) error {
	pid, err := s.readPIDFromFile()
	if err != nil {
		return fmt.Errorf("failed to read pid from file: %w", err)
	}

	proc, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil
		}

		return fmt.Errorf("failed to get process: %w", err)
	}

	if err := proc.TerminateWithContext(ctx); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	return nil
}

// PostDown removes PID file after the process has stopped.
func (s *Server) PostDown() error {
	if err := utils.RemoveFile(s.pidFilePath()); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

// AddPeer creates and registers a new VPN peer by issuing a new certificate.
func (s *Server) AddPeer(_ context.Context, req interface{}) (interface{}, error) {
	// Parse the request to ServiceRequest type.
	r, err := parseServiceRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	name := r.UUID.String()

	keyDER, certDER, err := s.pki.Issue(name)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	return &AddPeerResponse{
		Metadata: s.metadata,
		Cert:     certDER,
		Key:      keyDER,
	}, nil
}

// HasPeer checks if a peer is currently connected and tracked.
func (s *Server) HasPeer(ctx context.Context, req interface{}) (bool, error) {
	// Parse the request to ServiceRequest type.
	r, err := parseServiceRequest(req)
	if err != nil {
		return false, fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return false, fmt.Errorf("invalid request: %w", err)
	}

	items, err := s.PeerStatistics(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get peer statistics: %w", err)
	}

	name := r.UUID.String()
	for _, item := range items {
		if item.Key == name {
			return true, nil
		}
	}

	return false, nil
}

// RemovePeer disconnects a VPN client and revokes its certificate.
func (s *Server) RemovePeer(_ context.Context, req interface{}) error {
	// Parse the request to ServiceRequest type.
	r, err := parseServiceRequest(req)
	if err != nil {
		return fmt.Errorf("failed to parse request: %w", err)
	}
	if err := r.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	conn, err := s.mgmtConn()
	if err != nil {
		return fmt.Errorf("failed to get management connection: %w", err)
	}

	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Wait for management interface banner
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read line: %w", err)
		}
		if strings.Contains(line, "OpenVPN Management Interface") {
			break
		}
	}

	// Issue kill command for the client
	name := r.UUID.String()
	if _, err := fmt.Fprintf(conn, "kill %s\n", name); err != nil {
		return fmt.Errorf("failed to write command: %w", err)
	}

	// Await confirmation
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read line: %w", err)
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SUCCESS") {
			break
		}
		if strings.HasPrefix(line, "ERROR") {
			return fmt.Errorf("failed to kill client: %s", line)
		}
	}

	// Revoke the certificate
	if err := s.pki.Revoke(name); err != nil {
		return fmt.Errorf("failed to revoke certificate: %w", err)
	}

	return nil
}

// PeerCount returns the number of active peers.
func (s *Server) PeerCount() int {
	return 0
}

// PeerStatistics queries the OpenVPN management interface and returns peer usage data.
func (s *Server) PeerStatistics(_ context.Context) (items []*types.PeerStatistic, err error) {
	conn, err := s.mgmtConn()
	if err != nil {
		return nil, fmt.Errorf("failed to get management connection: %w", err)
	}

	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Wait for the management interface welcome message
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read line: %w", err)
		}
		if strings.Contains(line, "OpenVPN Management Interface") {
			break
		}
	}

	// Request status output
	if _, err := fmt.Fprintf(conn, "status\n"); err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}

	// Parse client statistics
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read line: %w", err)
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

		uploadBytes, err := strconv.ParseInt(fields[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse upload bytes: %w", err)
		}
		downloadBytes, err := strconv.ParseInt(fields[6], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse download bytes: %w", err)
		}

		items = append(items, &types.PeerStatistic{
			Key:           fields[1],
			DownloadBytes: downloadBytes,
			UploadBytes:   uploadBytes,
		})
	}

	return items, nil
}
