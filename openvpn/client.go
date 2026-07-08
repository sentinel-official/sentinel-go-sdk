package openvpn

import (
	"bufio"
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

	"github.com/sentinel-official/sentinel-go-sdk/libs/encoding/pem"
	"github.com/sentinel-official/sentinel-go-sdk/process"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Client implements types.ClientService interface.
var _ types.ClientService = (*Client)(nil)

// Client represents an OpenVPN client instance.
type Client struct {
	*process.Manager // Embedded process manager for handling lifecycle.

	cfg     *ClientConfig // Configuration settings for the service.
	homeDir string        // Home directory of the service.

	cmd *exec.Cmd // Command to run the OpenVPN client.
}

// NewClient creates a new Client instance.
func NewClient(name, appDir string, cfg *ClientConfig) *Client {
	return &Client{
		Manager: process.NewManager(name),
		cfg:     cfg,
		homeDir: filepath.Join(appDir, "openvpn"),
	}
}

// Type returns the service type of the client.
func (c *Client) Type() types.ServiceType {
	return types.ServiceTypeOpenVPN
}

// IsRunning checks if the OpenVPN client process is running.
func (c *Client) IsRunning() (bool, error) {
	// Read PID from file.
	pid, err := c.readPID()
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

	if name != openVPN {
		return false, nil
	}

	return true, nil
}

// Init sets up service configuration, creating directories and writing defaults unless config exists.
func (c *Client) Init(force bool) error {
	// Create the home directory if it doesn't exist
	if err := os.MkdirAll(c.homeDir, 0700); err != nil {
		return fmt.Errorf("creating home directory %q: %w", c.homeDir, err)
	}

	// Construct the full path to the config file
	cfgFile := c.appConfigFile()

	// Check if the config file exists at the specified path
	exists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
	}

	// Write config only if file doesn't exist or force flag is enabled
	if !exists || force {
		if err := c.cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("writing app config file %q: %w", cfgFile, err)
		}
	}

	return nil
}

// Setup prepares the OpenVPN client service for operation.
func (c *Client) Setup(ctx context.Context) error {
	return c.Manager.Setup(ctx, func() error { //nolint:wrapcheck
		// Construct the full path to the config file
		cfgFile := c.appConfigFile()

		// Check if the config file exists at the specified path
		exists, err := utils.IsFileExists(cfgFile)
		if err != nil {
			return fmt.Errorf("checking if config file %q exists: %w", cfgFile, err)
		}

		// If the config file exists, proceed to read its contents
		if exists {
			if err := c.cfg.ReadAppConfig(cfgFile); err != nil {
				return fmt.Errorf("reading app config file %q: %w", cfgFile, err)
			}
		}

		c.cfg.PKIDir = filepath.Join(c.homeDir, "pki")

		// Validate the config
		if err := c.cfg.Validate(); err != nil {
			return fmt.Errorf("validating config: %w", err)
		}

		// Create the PKI directory.
		if err := os.MkdirAll(c.cfg.PKIDir, 0700); err != nil {
			return fmt.Errorf("creating pki directory %q: %w", c.cfg.PKIDir, err)
		}

		// Write the CA certificate file.
		caFile := filepath.Join(c.cfg.PKIDir, "ca.crt")
		if err := pem.WriteFile(caFile, pem.FormatBase64, pem.BlockTypeCertificate, c.cfg.CA); err != nil {
			return fmt.Errorf("writing ca certificate file %q: %w", caFile, err)
		}

		// Write the client certificate file.
		certFile := filepath.Join(c.cfg.PKIDir, "client.crt")
		if err := pem.WriteFile(certFile, pem.FormatBase64, pem.BlockTypeCertificate, c.cfg.Cert); err != nil {
			return fmt.Errorf("writing client certificate file %q: %w", certFile, err)
		}

		// Write the client private key file.
		keyFile := filepath.Join(c.cfg.PKIDir, "client.key")
		if err := pem.WriteFile(keyFile, pem.FormatBase64, pem.BlockTypePrivateKey, c.cfg.Key); err != nil {
			return fmt.Errorf("writing client key file %q: %w", keyFile, err)
		}

		// Write the TLS static key file.
		tlsFile := filepath.Join(c.cfg.PKIDir, "tls.key")
		if err := pem.WriteFile(tlsFile, pem.FormatHex, pem.BlockTypeOpenVPNStaticKeyV1, c.cfg.TLS); err != nil {
			return fmt.Errorf("writing tls key file %q: %w", tlsFile, err)
		}

		// Write configuration to file.
		cfgFile = c.serviceConfigFile()
		if err := c.cfg.WriteServiceConfig(cfgFile); err != nil {
			return fmt.Errorf("writing service config file %q: %w", cfgFile, err)
		}

		return nil
	})
}

// Start starts the OpenVPN client service.
func (c *Client) Start(parent context.Context) (context.Context, error) {
	return c.Manager.Start(parent, func(ctx context.Context) error { //nolint:wrapcheck
		// Constructs the command to start the OpenVPN client.
		cfgFile := c.serviceConfigFile()
		c.cmd = exec.CommandContext(
			ctx,
			c.execFile(openVPN),
			"--config", cfgFile,
		)
		c.cmd.Stderr = os.Stderr

		// Starts the OpenVPN client process.
		if err := c.cmd.Start(); err != nil {
			return fmt.Errorf("starting command: %w", err)
		}

		// Write PID to file.
		if err := c.writePID(c.cmd.Process.Pid); err != nil {
			return fmt.Errorf("writing PID: %w", err)
		}

		// Install the kill-switch firewall rules.
		if err := c.applyFirewall(ctx); err != nil {
			_ = c.cmd.Process.Kill()
			_ = c.cmd.Wait()

			return fmt.Errorf("applying firewall rules: %w", err)
		}

		// Wait for the OpenVPN process to finish in a separate goroutine.
		c.Go(ctx, func() (err error) {
			if err = c.cmd.Wait(); err == nil {
				err = errors.New("exited unexpectedly")
			}

			return fmt.Errorf("waiting command: %w", err)
		})

		// Set the system DNS once the TUN interface is up.
		c.Go(ctx, func() error {
			if err := c.applyDNS(ctx); err != nil && ctx.Err() == nil {
				return fmt.Errorf("applying dns: %w", err)
			}

			return nil
		})

		return nil
	})
}

// Stop stops the OpenVPN client service.
func (c *Client) Stop() error {
	return c.Manager.Stop(func() error { //nolint:wrapcheck
		// Revert the system DNS (best-effort).
		if err := c.removeDNS(context.Background()); err != nil {
			return fmt.Errorf("removing dns: %w", err)
		}

		// Remove the kill-switch firewall rules (best-effort).
		if err := c.removeFirewall(context.Background()); err != nil {
			return fmt.Errorf("removing firewall rules: %w", err)
		}

		// Read PID from file.
		pid, err := c.readPID()
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
func (c *Client) Wait(ctx context.Context) error {
	return c.Manager.Wait(ctx, nil) //nolint:wrapcheck
}

// Cleanup removes service configuration files.
func (c *Client) Cleanup() error {
	return c.Manager.Cleanup(func() error { //nolint:wrapcheck
		// Removes configuration file.
		cfgFile := c.serviceConfigFile()
		if err := utils.RemoveFile(cfgFile); err != nil {
			return fmt.Errorf("removing config file %q: %w", cfgFile, err)
		}

		// Remove PID file.
		pidFile := c.pidFile()
		if err := utils.RemoveFile(pidFile); err != nil {
			return fmt.Errorf("removing PID file %q: %w", pidFile, err)
		}

		return nil
	})
}

// Statistics retrieves the download and upload statistics from the OpenVPN client.
func (c *Client) Statistics(ctx context.Context) (int64, int64, error) {
	conn, err := c.mgmtConn(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("getting management connection: %w", err)
	}

	defer func() {
		_ = conn.Close()
	}()

	reader := bufio.NewReader(conn)

	// Wait for the management interface banner.
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, 0, fmt.Errorf("reading line: %w", err)
		}

		if strings.Contains(line, "OpenVPN Management Interface") {
			break
		}
	}

	// Request load stats.
	if _, err := fmt.Fprintf(conn, "load-stats\n"); err != nil {
		return 0, 0, fmt.Errorf("writing load-stats command: %w", err)
	}

	// Parse response.
	var rxBytes, txBytes int64

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, 0, fmt.Errorf("reading line: %w", err)
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ERROR") {
			return 0, 0, fmt.Errorf("reading load stats: %s", line)
		}

		if !strings.HasPrefix(line, "SUCCESS:") {
			continue
		}

		for field := range strings.SplitSeq(strings.TrimPrefix(line, "SUCCESS:"), ",") {
			key, value, ok := strings.Cut(strings.TrimSpace(field), "=")
			if !ok {
				continue
			}

			switch key {
			case "bytesin":
				v, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return 0, 0, fmt.Errorf("parsing bytesin %q: %w", value, err)
				}

				rxBytes = v
			case "bytesout":
				v, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return 0, 0, fmt.Errorf("parsing bytesout %q: %w", value, err)
				}

				txBytes = v
			}
		}

		return rxBytes, txBytes, nil
	}
}

func (c *Client) appConfigFile() string     { return filepath.Join(c.homeDir, "config.toml") }
func (c *Client) pidFile() string           { return filepath.Join(c.homeDir, "client.pid") }
func (c *Client) serviceConfigFile() string { return filepath.Join(c.homeDir, "client.conf") }

// readPID reads the PID from the client's PID file.
func (c *Client) readPID() (int32, error) {
	// Get the full path to the PID file
	pidFile := filepath.Clean(c.pidFile())

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

// writePID writes the given PID to the client's PID file.
func (c *Client) writePID(pid int) error {
	// Convert PID to byte slice.
	data := []byte(strconv.Itoa(pid))

	// Write PID to file with appropriate permissions.
	pidFile := c.pidFile()
	if err := os.WriteFile(pidFile, data, 0600); err != nil {
		return fmt.Errorf("writing PID file %q: %w", pidFile, err)
	}

	return nil
}

// mgmtConn creates a TCP connection to the OpenVPN management interface.
func (c *Client) mgmtConn(ctx context.Context) (_ net.Conn, err error) {
	target := "127.0.0.1:2323"
	timeout := 5 * time.Second

	// Create a Dialer with timeout
	dialer := &net.Dialer{Timeout: timeout}

	// Create the connection with context-aware dialer
	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return nil, fmt.Errorf("creating connection for target %q: %w", target, err)
	}

	// Ensure the connection is closed if an error occurs
	defer func() {
		if err != nil && conn != nil {
			_ = conn.Close()
		}
	}()

	// Set the deadline immediately after connection is established
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("setting connection deadline %q: %w", deadline, err)
	}

	return conn, nil
}
