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

	"github.com/shirou/gopsutil/v4/process"
	proxymancommand "github.com/v2fly/v2ray-core/v5/app/proxyman/command"
	statscommand "github.com/v2fly/v2ray-core/v5/app/stats/command"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Server implements types.ServerService interface.
var _ types.ServerService = (*Server)(nil)

// Server represents the V2Ray server instance.
type Server struct {
	cmd      *exec.Cmd         // Command to run the V2Ray server.
	homeDir  string            // Home directory of the V2Ray server.
	metadata []*ServerMetadata // Metadata for server's inbound connections.
	name     string            // Name of the server instance.
	pm       *PeerManager      // Peer manager for handling peer information.
}

// NewServer creates a new Server instance.
func NewServer() *Server {
	return &Server{}
}

// WithHomeDir sets the home directory for the server and returns the updated Server instance.
func (s *Server) WithHomeDir(homeDir string) *Server {
	s.homeDir = homeDir
	return s
}

// WithName sets the name for the server and returns the updated Server instance.
func (s *Server) WithName(name string) *Server {
	s.name = name
	return s
}

// WithPeerManager sets the PeerManager for the server and returns the updated Server instance.
func (s *Server) WithPeerManager(pm *PeerManager) *Server {
	s.pm = pm
	return s
}

// configFilePath returns the full path of the V2Ray server's configuration file.
func (s *Server) configFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.json", s.name))
}

// pidFilePath returns the file path of the server's PID file.
func (s *Server) pidFilePath() string {
	return filepath.Join(s.homeDir, fmt.Sprintf("%s.pid", s.name))
}

// readPIDFromFile reads the PID from the server's PID file.
func (s *Server) readPIDFromFile() (int32, error) {
	name := s.pidFilePath()
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return 0, nil
	}

	// Read PID from the PID file.
	data, err := os.ReadFile(name)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	// Convert PID data to integer.
	pid, err := strconv.ParseInt(string(data), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse pid: %w", err)
	}

	return int32(pid), nil
}

// writePIDToFile writes the given PID to the server's PID file.
func (s *Server) writePIDToFile(pid int) error {
	// Convert PID to byte slice.
	data := []byte(strconv.Itoa(pid))

	// Write PID to file with appropriate permissions.
	if err := os.WriteFile(s.pidFilePath(), data, 0644); err != nil {
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

// IsUp checks if the V2Ray server process is running.
func (s *Server) IsUp(ctx context.Context) (bool, error) {
	// Read PID from file.
	pid, err := s.readPIDFromFile()
	if err != nil {
		return false, fmt.Errorf("failed to read pid from file: %w", err)
	}
	if pid == 0 {
		return false, nil
	}

	// Retrieve process with the given PID.
	proc, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get process: %w", err)
	}

	// Check if the process is running.
	ok, err := proc.IsRunningWithContext(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check running process: %w", err)
	}
	if !ok {
		return false, nil
	}

	// Retrieve the name of the process.
	name, err := proc.NameWithContext(ctx)
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
func (s *Server) PreUp(v interface{}) error {
	// Check for valid parameter type.
	cfg, ok := v.(*ServerConfig)
	if !ok {
		return fmt.Errorf("invalid parameter type %T", v)
	}

	for _, inbound := range cfg.Inbounds {
		metadata := &ServerMetadata{
			Tag: inbound.Tag(),
		}

		s.metadata = append(s.metadata, metadata)
	}

	// Write configuration to file.
	if err := cfg.WriteToFile(s.configFilePath()); err != nil {
		return fmt.Errorf("failed to write config to file: %w", err)
	}

	return nil
}

// Up starts the V2Ray server process.
func (s *Server) Up(ctx context.Context) error {
	// Constructs the command to start the V2Ray server.
	s.cmd = exec.CommandContext(
		ctx,
		s.execFile(v2ray),
		strings.Fields(fmt.Sprintf("run --config %s", s.configFilePath()))...,
	)

	// Starts the V2Ray server process.
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	return nil
}

// PostUp performs operations after the server process is started.
func (s *Server) PostUp() error {
	// Check if command or process is nil.
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("nil command or process")
	}

	// Write PID to file.
	if err := s.writePIDToFile(s.cmd.Process.Pid); err != nil {
		return fmt.Errorf("failed to write pid to file: %w", err)
	}

	if err := s.cmd.Wait(); err != nil {
		return fmt.Errorf("failed to wait for command: %w", err)
	}

	return nil
}

// PreDown performs operations before the server process is terminated.
func (s *Server) PreDown() error {
	return nil
}

// Down terminates the V2Ray server process.
func (s *Server) Down(ctx context.Context) error {
	// Read PID from file.
	pid, err := s.readPIDFromFile()
	if err != nil {
		return fmt.Errorf("failed to read pid from file: %w", err)
	}

	// Retrieve process with the given PID.
	proc, err := process.NewProcessWithContext(ctx, pid)
	if err != nil {
		if errors.Is(err, process.ErrorProcessNotRunning) {
			return nil
		}

		return fmt.Errorf("failed to get process: %w", err)
	}

	// Terminate the process.
	if err := proc.TerminateWithContext(ctx); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	return nil
}

// PostDown performs cleanup operations after the server process is terminated.
func (s *Server) PostDown() error {
	// Remove PID file.
	if err := utils.RemoveFile(s.pidFilePath()); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

// AddPeer adds a new peer to the V2Ray server.
func (s *Server) AddPeer(ctx context.Context, req interface{}) (interface{}, error) {
	// Cast the request to AddPeerRequest type.
	r, ok := req.(*AddPeerRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type: %T", req)
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Establish a gRPC client connection to the handler service.
	conn, client, err := s.handlerServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get handler service client: %w", err)
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Extract key from the request.
	email := r.Key()

	for _, md := range s.metadata {
		// Prepare gRPC request to add a new user to the handler.
		in := &proxymancommand.AlterInboundRequest{
			Tag: md.Tag.String(),
			Operation: serial.ToTypedMessage(
				&proxymancommand.AddUserOperation{
					User: &protocol.User{
						Email:   email,
						Account: md.Tag.Account(r.UUID),
					},
				},
			),
		}

		// Send the request to add a user to the handler.
		if _, err := client.AlterInbound(ctx, in); err != nil {
			return nil, fmt.Errorf("failed to alter inbound: %w", err)
		}
	}

	// Update the local peer collection with the new peer information.
	s.pm.Put(
		&Peer{
			Email: email,
		},
	)

	// Return nil for success (no additional data to return in response).
	return &AddPeerResponse{
		Metadata: s.metadata,
	}, nil
}

// HasPeer checks if a peer exists in the V2Ray server's peer list.
func (s *Server) HasPeer(_ context.Context, req interface{}) (bool, error) {
	// Cast the request to HasPeerRequest type.
	r, ok := req.(*HasPeerRequest)
	if !ok {
		return false, fmt.Errorf("invalid request type: %T", req)
	}
	if err := r.Validate(); err != nil {
		return false, fmt.Errorf("invalid request: %w", err)
	}

	// Retrieve the key from the request.
	email := r.Key()
	peer := s.pm.Get(email)

	// Return true if the peer exists, otherwise false.
	return peer != nil, nil
}

// RemovePeer removes a peer from the V2Ray server.
func (s *Server) RemovePeer(ctx context.Context, req interface{}) error {
	// Cast the request to RemovePeerRequest type.
	r, ok := req.(*RemovePeerRequest)
	if !ok {
		return fmt.Errorf("invalid request type: %T", req)
	}
	if err := r.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// Establish a gRPC client connection to the handler service.
	conn, client, err := s.handlerServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get handler service client: %w", err)
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Extract key from the request.
	email := r.Key()

	for _, md := range s.metadata {
		// Prepare gRPC request to remove a user from the handler.
		in := &proxymancommand.AlterInboundRequest{
			Tag: md.Tag.String(),
			Operation: serial.ToTypedMessage(
				&proxymancommand.RemoveUserOperation{
					Email: email,
				},
			),
		}

		// Send the request to remove a user from the handler.
		if _, err := client.AlterInbound(ctx, in); err != nil {
			// If the user is not found, continue without error.
			if !strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("failed to alter inbound: %w", err)
			}
		}
	}

	// Remove the peer information from the local collection.
	s.pm.Delete(email)

	// Return nil for success.
	return nil
}

// PeerCount returns the number of peers connected to the V2Ray server.
func (s *Server) PeerCount() int {
	return s.pm.Len()
}

// PeerStatistics retrieves statistics for each peer connected to the V2Ray server.
func (s *Server) PeerStatistics(ctx context.Context) (items []*types.PeerStatistic, err error) {
	// Establish a gRPC client connection to the stats service.
	conn, client, err := s.statsServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats service client: %w", err)
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Define a function to process each peer in the local collection.
	fn := func(key string, _ *Peer) (bool, error) {
		// Prepare gRPC request to get uplink traffic stats.
		in := &statscommand.GetStatsRequest{
			Reset_: false,
			Name:   fmt.Sprintf("user>>>%s>>>traffic>>>uplink", key),
		}

		// Send the request to get uplink traffic stats.
		res, err := client.GetStats(ctx, in)
		if err != nil {
			// If the stat is not found, continue to the next peer.
			if !strings.Contains(err.Error(), "not found") {
				return false, fmt.Errorf("failed to get stats: %w", err)
			}
		}

		// Extract uplink traffic stats or use an empty stat if not found.
		upLink := res.GetStat()
		if upLink == nil {
			upLink = &statscommand.Stat{}
		}

		// Prepare gRPC request to get downlink traffic stats.
		in = &statscommand.GetStatsRequest{
			Reset_: false,
			Name:   fmt.Sprintf("user>>>%s>>>traffic>>>downlink", key),
		}

		// Send the request to get downlink traffic stats.
		res, err = client.GetStats(ctx, in)
		if err != nil {
			// If the stat is not found, continue to the next peer.
			if !strings.Contains(err.Error(), "not found") {
				return false, fmt.Errorf("failed to get stats: %w", err)
			}
		}

		// Extract downlink traffic stats or use an empty stat if not found.
		downLink := res.GetStat()
		if downLink == nil {
			downLink = &statscommand.Stat{}
		}

		// Append peer statistics to the result collection.
		items = append(
			items,
			&types.PeerStatistic{
				Key:           key,
				DownloadBytes: downLink.GetValue(),
				UploadBytes:   upLink.GetValue(),
			},
		)

		return false, nil
	}

	// Iterate over each peer and retrieve statistics.
	if err := s.pm.Iterate(fn); err != nil {
		return nil, fmt.Errorf("failed to iterate peers: %w", err)
	}

	// Return the constructed collection of peer statistics.
	return items, nil
}
