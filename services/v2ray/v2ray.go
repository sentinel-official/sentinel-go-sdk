package v2ray

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	proxymancommand "github.com/v2fly/v2ray-core/v5/app/proxyman/command"
	statscommand "github.com/v2fly/v2ray-core/v5/app/stats/command"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/common/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sentinel-official/sentinel-go-sdk/v1/services/v2ray/types"
	sentinelsdk "github.com/sentinel-official/sentinel-go-sdk/v1/types"
)

const (
	// DataLen represents the expected length of data used for peer operations.
	DataLen = 1 + 16

	// InfoLen represents the length of the server information.
	InfoLen = 2 + 1

	// ConfigFilename represents the name of the configuration file.
	ConfigFilename = "v2ray_config.json"
)

var (
	_ sentinelsdk.ClientService = (*Client)(nil)
	_ sentinelsdk.ServerService = (*Server)(nil)
)

type Client struct {
}

func (c *Client) Down() error {
	// TODO implement me
	panic("implement me")
}

func (c *Client) Info() []byte {
	// TODO implement me
	panic("implement me")
}

func (c *Client) IsUp() bool {
	// TODO implement me
	panic("implement me")
}

func (c *Client) PostDown() error {
	// TODO implement me
	panic("implement me")
}

func (c *Client) PostUp() error {
	// TODO implement me
	panic("implement me")
}

func (c *Client) PreDown() error {
	// TODO implement me
	panic("implement me")
}

func (c *Client) PreUp() error {
	// TODO implement me
	panic("implement me")
}

func (c *Client) Statistics() (int64, int64, error) {
	// TODO implement me
	panic("implement me")
}

func (c *Client) Up() error {
	// TODO implement me
	panic("implement me")
}

// Server represents the V2Ray server instance.
type Server struct {
	cmd     *exec.Cmd    // cmd is the command for running the V2Ray server.
	homeDir string       // homeDir is the home directory of the V2Ray server.
	info    []byte       // info stores information about the server.
	peers   *types.Peers // peers is a collection of peer information.
}

// NewServer creates a new instance of the V2Ray server.
func NewServer(homeDir string) *Server {
	return &Server{
		cmd:     nil,
		homeDir: homeDir,
		info:    make([]byte, InfoLen),
		peers:   types.NewPeers(),
	}
}

// clientConn establishes a gRPC client connection to the V2Ray server.
func (s *Server) clientConn() (*grpc.ClientConn, error) {
	// Define the target address for the gRPC client connection.
	target := "127.0.0.1:23"

	// Establish a gRPC client connection with specified options:
	// - WithBlock: Blocks until the underlying connection is established.
	// - WithTransportCredentials: Configures insecure transport credentials for the connection.
	return grpc.Dial(
		target,
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

// configFilePath returns the full path of the V2Ray server's configuration file.
func (s *Server) configFilePath() string {
	return filepath.Join(s.homeDir, ConfigFilename)
}

// handlerServiceClient establishes a gRPC client connection to the V2Ray server's handler service.
func (s *Server) handlerServiceClient() (*grpc.ClientConn, proxymancommand.HandlerServiceClient, error) {
	// Establish a gRPC client connection using the clientConn method.
	conn, err := s.clientConn()
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	// Create a new StatsServiceClient using the established connection.
	client := statscommand.NewStatsServiceClient(conn)

	// Return both the connection and the client.
	return conn, client, nil
}

// AddPeer adds a new peer to the V2Ray server.
func (s *Server) AddPeer(ctx context.Context, buf []byte) ([]byte, error) {
	// Check if the data length is valid.
	if len(buf) != DataLen {
		return nil, fmt.Errorf("invalid data length; expected %d, got %d", DataLen, len(buf))
	}

	// Establish a gRPC client connection to the handler service.
	conn, client, err := s.handlerServiceClient()
	if err != nil {
		return nil, err
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Encode the data buffer to email using base64 encoding and extract proxy type.
	var (
		email = base64.StdEncoding.EncodeToString(buf)
		proxy = types.Proxy(buf[0])
	)

	// Parse the UUID from the data buffer.
	uid, err := uuid.ParseBytes(buf[1:])
	if err != nil {
		return nil, err
	}

	// Prepare gRPC request to add a user to the handler.
	req := &proxymancommand.AlterInboundRequest{
		Tag: proxy.Tag(),
		Operation: serial.ToTypedMessage(
			&proxymancommand.AddUserOperation{
				User: &protocol.User{
					Level:   0,
					Email:   email,
					Account: proxy.Account(uid),
				},
			},
		),
	}

	// Send the request to add a user to the handler.
	_, err = client.AlterInbound(ctx, req)
	if err != nil {
		return nil, err
	}

	// Update the local peer collection with the new peer information.
	s.peers.Put(
		&types.Peer{
			Email: email,
		},
	)

	// Return nil for success (no additional data to return in response).
	return nil, nil
}

// HasPeer checks if a peer exists in the V2Ray server's peer list.
func (s *Server) HasPeer(_ context.Context, buf []byte) (bool, error) {
	// Check if the data length is valid.
	if len(buf) != DataLen {
		return false, fmt.Errorf("invalid data length; expected %d, got %d", DataLen, len(buf))
	}

	// Encode the data buffer to email using base64 encoding.
	var (
		email = base64.StdEncoding.EncodeToString(buf)
		peer  = s.peers.Get(email)
	)

	// Return true if the peer exists, otherwise false.
	return peer != nil, nil
}

// Info returns information about the V2Ray server.
func (s *Server) Info() []byte {
	return s.info
}

// Init initializes the V2Ray server.
func (s *Server) Init() error {
	// TODO: Initialization logic.
	// ...

	return nil
}

// PeerCount returns the number of peers connected to the V2Ray server.
func (s *Server) PeerCount() int {
	return s.peers.Len()
}

// PeerStatistics retrieves statistics for each peer connected to the V2Ray server.
func (s *Server) PeerStatistics(ctx context.Context) (items []*sentinelsdk.PeerStatistic, err error) {
	// Establish a gRPC client connection to the stats service.
	conn, client, err := s.statsServiceClient()
	if err != nil {
		return nil, err
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Define a function to process each peer in the local collection.
	fn := func(key string, _ *types.Peer) (bool, error) {
		// Prepare gRPC request to get uplink traffic stats.
		req := &statscommand.GetStatsRequest{
			Reset_: false,
			Name:   fmt.Sprintf("user>>>%s>>>traffic>>>uplink", key),
		}

		// Send the request to get uplink traffic stats.
		res, err := client.GetStats(ctx, req)
		if err != nil {
			// If the stat is not found, continue to the next peer.
			if !strings.Contains(err.Error(), "not found") {
				return false, err
			}
		}

		// Extract uplink traffic stats or use an empty stat if not found.
		upLink := res.GetStat()
		if upLink == nil {
			upLink = &statscommand.Stat{}
		}

		// Prepare gRPC request to get downlink traffic stats.
		req = &statscommand.GetStatsRequest{
			Reset_: false,
			Name:   fmt.Sprintf("user>>>%s>>>traffic>>>downlink", key),
		}

		// Send the request to get downlink traffic stats.
		res, err = client.GetStats(ctx, req)
		if err != nil {
			// If the stat is not found, continue to the next peer.
			if !strings.Contains(err.Error(), "not found") {
				return false, err
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
			&sentinelsdk.PeerStatistic{
				Key:      key,
				Upload:   upLink.GetValue(),
				Download: downLink.GetValue(),
			},
		)

		return false, nil
	}

	// Iterate over each peer and retrieve statistics.
	if err := s.peers.Iterate(fn); err != nil {
		return nil, err
	}

	// Return the constructed collection of peer statistics.
	return items, nil
}

// RemovePeer removes a peer from the V2Ray server.
func (s *Server) RemovePeer(ctx context.Context, buf []byte) error {
	// Check if the data length is valid.
	if len(buf) != DataLen {
		return fmt.Errorf("invalid data length; expected %d, got %d", DataLen, len(buf))
	}

	// Establish a gRPC client connection to the handler service.
	conn, client, err := s.handlerServiceClient()
	if err != nil {
		return err
	}

	// Ensure the connection is closed when done.
	defer func() {
		if err = conn.Close(); err != nil {
			panic(err)
		}
	}()

	// Encode the data buffer to email using base64 encoding and extract proxy type.
	var (
		email = base64.StdEncoding.EncodeToString(buf)
		proxy = types.Proxy(buf[0])
	)

	// Prepare gRPC request to remove a user from the handler.
	req := &proxymancommand.AlterInboundRequest{
		Tag: proxy.Tag(),
		Operation: serial.ToTypedMessage(
			&proxymancommand.RemoveUserOperation{
				Email: email,
			},
		),
	}

	// Send the request to remove a user from the handler.
	_, err = client.AlterInbound(ctx, req)
	if err != nil {
		// If the user is not found, continue without error.
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
	}

	// Remove the peer information from the local collection.
	s.peers.Delete(email)

	// Return nil for success.
	return nil
}

// Start starts the V2Ray server.
func (s *Server) Start() error {
	// Create a new command to execute the V2Ray binary.
	s.cmd = exec.Command(
		s.execFile(), // Get the path to the V2Ray binary.
		strings.Split(
			fmt.Sprintf("run --config %s", s.configFilePath()), // Construct the command-line arguments.
			" ",
		)...,
	)

	// Redirect standard output and error streams to the console.
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr

	// Start the V2Ray server by executing the command.
	return s.cmd.Start()
}

// Stop stops the V2Ray server.
func (s *Server) Stop() error {
	// Check if the command is nil.
	if s.cmd == nil {
		// If the command is nil, return an error indicating that the command is not initialized.
		return errors.New("nil cmd")
	}

	// Kill the process associated with the command to stop the V2Ray server.
	return s.cmd.Process.Kill()
}

// Type returns the service type of the V2Ray server.
func (s *Server) Type() sentinelsdk.ServiceType {
	return sentinelsdk.ServiceTypeV2Ray
}
