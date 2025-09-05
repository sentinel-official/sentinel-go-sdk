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

	procutils "github.com/shirou/gopsutil/v4/process"

	"github.com/sentinel-official/sentinel-go-sdk/process"
	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Client implements types.ClientService interface.
var _ types.ClientService = (*Client)(nil)

// Client represents the V2Ray client instance.
type Client struct {
	*process.Manager // Embedded process manager for handling lifecycle.

	cfg     *ClientConfig // Configuration settings for the service.
	homeDir string        // Home directory of the service.

	cmd *exec.Cmd // Command to run the V2Ray client.
}

// NewClient creates a new Client instance.
func NewClient(ctx context.Context, name, appDir string, cfg *ClientConfig) *Client {
	return &Client{
		Manager: process.NewManager(ctx, name),
		cfg:     cfg,
		homeDir: filepath.Join(appDir, "v2ray"),
	}
}

func (c *Client) appConfigFile() string     { return filepath.Join(c.homeDir, "config.toml") }
func (c *Client) pidFile() string           { return filepath.Join(c.homeDir, "client.pid") }
func (c *Client) serviceConfigFile() string { return filepath.Join(c.homeDir, "client.json") }

// readPID reads the PID from the client's PID file.
func (c *Client) readPID() (int32, error) {
	// Get the full path to the PID file
	pidFile := c.pidFile()

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

// Type returns the service type of the client.
func (c *Client) Type() types.ServiceType {
	return types.ServiceTypeV2Ray
}

// IsRunning checks if the V2Ray client process is running.
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
	if name != v2ray {
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

// Start starts the V2Ray client service.
func (c *Client) Start() error {
	return c.Manager.Start(func(ctx context.Context) error {
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

		// Constructs the command to start the V2Ray client.
		c.cmd = exec.CommandContext(
			ctx,
			c.execFile(v2ray),
			strings.Fields(fmt.Sprintf("run --config %s", cfgFile))...,
		)

		// Starts the V2Ray client process.
		if err := c.cmd.Start(); err != nil {
			return fmt.Errorf("starting command: %w", err)
		}

		// Write PID to file.
		if err := c.writePID(c.cmd.Process.Pid); err != nil {
			return fmt.Errorf("writing PID: %w", err)
		}

		// Wait for the V2Ray process to finish in a separate goroutine.
		c.Go(func(ctx context.Context) (err error) {
			if err = c.cmd.Wait(); err == nil {
				err = errors.New("exited unexpectedly")
			}

			return fmt.Errorf("waiting command: %w", err)
		})

		return nil
	})
}

// Stop stops the V2Ray client service.
func (c *Client) Stop() error {
	return c.Manager.Stop(func() error {
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
func (c *Client) Wait() error {
	return c.Manager.Wait(nil)
}

// Cleanup removes service configuration files.
func (c *Client) Cleanup() error {
	return c.Manager.Cleanup(func() error {
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

// Statistics returns dummy statistics for now (to be implemented).
func (c *Client) Statistics(_ context.Context) (int64, int64, error) {
	return 0, 0, errors.New("not implemented")
}
