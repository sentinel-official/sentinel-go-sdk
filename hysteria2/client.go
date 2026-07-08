package hysteria2

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	netutils "github.com/shirou/gopsutil/v4/net"
	procutils "github.com/shirou/gopsutil/v4/process"

	"github.com/sentinel-official/sentinel-go-sdk/process"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Client implements types.ClientService interface.
var _ types.ClientService = (*Client)(nil)

// Client represents the Hysteria2 client instance.
type Client struct {
	*process.Manager // Embedded process manager for handling lifecycle.

	cfg     *ClientConfig // Configuration settings for the service.
	homeDir string        // Home directory of the service.

	cmd *exec.Cmd // Command to run the Hysteria2 client.
}

// NewClient creates a new Client instance.
func NewClient(name, appDir string, cfg *ClientConfig) *Client {
	return &Client{
		Manager: process.NewManager(name),
		cfg:     cfg,
		homeDir: filepath.Join(appDir, hysteria2),
	}
}

// Type returns the service type of the client.
func (c *Client) Type() types.ServiceType {
	return types.ServiceTypeHysteria2
}

// IsRunning checks if the Hysteria2 client process is running.
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

	if name != hysteria2 {
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

// Setup prepares the Hysteria2 client service for operation.
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

		// Validate the config
		if err := c.cfg.Validate(); err != nil {
			return fmt.Errorf("validating config: %w", err)
		}

		// Write configuration to file.
		cfgFile = c.serviceConfigFile()
		if err := c.cfg.WriteServiceConfig(cfgFile); err != nil {
			return fmt.Errorf("writing service config file %q: %w", cfgFile, err)
		}

		return nil
	})
}

// Start starts the Hysteria2 client service.
func (c *Client) Start(parent context.Context) (context.Context, error) {
	return c.Manager.Start(parent, func(ctx context.Context) error { //nolint:wrapcheck
		// Constructs the command to start the Hysteria2 client.
		cfgFile := c.serviceConfigFile()
		c.cmd = exec.CommandContext(
			ctx,
			c.execFile(hysteria2),
			"client", "-c", cfgFile,
		)
		c.cmd.Stderr = os.Stderr

		// Starts the Hysteria2 client process.
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

		// Wait for the Hysteria2 process to finish in a separate goroutine.
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

// Stop stops the Hysteria2 client service.
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

// Statistics returns the download and upload byte counts for the TUN interface.
func (c *Client) Statistics(ctx context.Context) (int64, int64, error) {
	return c.ifaceStats(ctx, c.cfg.TUNIface)
}

// ifaceStats reads the TUN interface rx/tx byte counters using OS network I/O counters.
// Host RX maps to download (BytesRecv), host TX maps to upload (BytesSent).
func (c *Client) ifaceStats(ctx context.Context, iface string) (int64, int64, error) {
	// Retrieve per-interface I/O counters.
	counters, err := netutils.IOCountersWithContext(ctx, true)
	if err != nil {
		return 0, 0, fmt.Errorf("reading interface counters: %w", err)
	}

	// Find the entry matching the TUN interface name.
	for _, counter := range counters {
		if counter.Name != iface {
			continue
		}

		return int64(counter.BytesRecv), int64(counter.BytesSent), nil
	}

	return 0, 0, fmt.Errorf("interface %q not found", iface)
}

func (c *Client) appConfigFile() string     { return filepath.Join(c.homeDir, "config.toml") }
func (c *Client) pidFile() string           { return filepath.Join(c.homeDir, "client.pid") }
func (c *Client) serviceConfigFile() string { return filepath.Join(c.homeDir, "client.yaml") }

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
