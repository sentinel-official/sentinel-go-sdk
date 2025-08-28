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
	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Client implements the types.ClientService interface.
var _ types.ClientService = (*Client)(nil)

// Client represents a V2Ray client with associated command, home directory, and name.
type Client struct {
	cfg     *ClientConfig // Configuration settings for the V2Ray client.
	cmd     *exec.Cmd     // Command for running the V2Ray client.
	homeDir string        // Home directory for client files.
	name    string        // Name of the interface.

	cancel context.CancelFunc // Context cancel function to stop background tasks.
	ctx    context.Context    // Context for client lifecycle management.
	eg     *errgroup.Group    // Error group for managing background goroutines.
}

// NewClient creates a new Client instance.
func NewClient(appDir string, cfg *ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Client{
		cfg:     cfg,
		homeDir: filepath.Join(appDir, "v2ray"),
		name:    "client",
		cancel:  cancel,
		ctx:     ctx,
		eg:      eg,
	}
}

// WithName sets the name for the client and returns the updated Client instance.
func (c *Client) WithName(name string) *Client {
	c.name = name
	return c
}

// appConfigFilePath returns the full path to the application's configuration file.
func (c *Client) appConfigFilePath() string {
	return filepath.Join(c.homeDir, "config.toml")
}

// serviceConfigFilePath returns the full path to the service-specific configuration file.
func (c *Client) serviceConfigFilePath() string {
	return filepath.Join(c.homeDir, fmt.Sprintf("%s.json", c.name))
}

// pidFilePath returns the file path of the client's PID file.
func (c *Client) pidFilePath() string {
	return filepath.Join(c.homeDir, fmt.Sprintf("%s.pid", c.name))
}

// readPIDFromFile reads the PID from the client's PID file.
func (c *Client) readPIDFromFile() (int32, error) {
	// Get the full path to the PID file
	pidFile := c.pidFilePath()

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

// writePIDToFile writes the given PID to the client's PID file.
func (c *Client) writePIDToFile(pid int) error {
	// Convert PID to byte slice.
	data := []byte(strconv.Itoa(pid))

	// Write PID to file with appropriate permissions.
	pidFile := c.pidFilePath()
	if err := os.WriteFile(pidFile, data, 0644); err != nil {
		return fmt.Errorf("writing PID file %q: %w", pidFile, err)
	}

	return nil
}

// Type returns the service type of the client.
func (c *Client) Type() types.ServiceType {
	return types.ServiceTypeV2Ray
}

// Init sets up service configuration, creating directories and writing defaults unless config exists.
func (c *Client) Init(force bool) error {
	// Create the home directory if it doesn't exist
	if err := os.MkdirAll(c.homeDir, 0755); err != nil {
		return fmt.Errorf("creating home directory %q: %w", c.homeDir, err)
	}

	// Construct the full path to the config file
	cfgFile := c.appConfigFilePath()

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

// IsUp checks if the V2Ray client process is running.
func (c *Client) IsUp() (bool, error) {
	// Read PID from file.
	pid, err := c.readPIDFromFile()
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

	// Check if the process name matches constant v2ray.
	if name != v2ray {
		return false, nil
	}

	return true, nil
}

// PreUp writes the configuration to the config file before starting the client process.
func (c *Client) PreUp() error {
	// Construct the full path to the config file
	cfgFile := c.appConfigFilePath()

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
	cfgFile = c.serviceConfigFilePath()
	if err := c.cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("writing service config file %q: %w", cfgFile, err)
	}

	return nil
}

// Up starts the V2Ray client process.
func (c *Client) Up() error {
	// Constructs the command to start the V2Ray client.
	cfgFile := c.serviceConfigFilePath()
	c.cmd = exec.CommandContext(
		c.ctx,
		c.execFile(v2ray),
		strings.Fields(fmt.Sprintf("run --config %s", cfgFile))...,
	)

	// Starts the V2Ray client process.
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %w", err)
	}

	// Wait for the V2Ray process to finish in a separate goroutine.
	c.eg.Go(func() (err error) {
		if err = c.cmd.Wait(); err == nil {
			err = errors.New("command exited unexpectedly")
		}

		return fmt.Errorf("waiting command: %w", err)
	})

	return nil
}

// PostUp performs operations after the client process is started.
func (c *Client) PostUp() error {
	// Write PID to file.
	if err := c.writePIDToFile(c.cmd.Process.Pid); err != nil {
		return fmt.Errorf("writing PID to file: %w", err)
	}

	return nil
}

// Wait blocks until all goroutines in the error group finish or one returns an error.
func (c *Client) Wait() error {
	if err := c.eg.Wait(); err != nil {
		return err
	}

	return nil
}

// PreDown performs operations before the client process is terminated.
func (c *Client) PreDown() error {
	// Cancel background tasks if any.
	c.cancel()

	return nil
}

// Down terminates the V2Ray client process.
func (c *Client) Down() error {
	// Read PID from file.
	pid, err := c.readPIDFromFile()
	if err != nil {
		return fmt.Errorf("reading PID from file: %w", err)
	}
	if pid == 0 {
		return nil
	}

	// Retrieve process with the given PID.
	proc, err := process.NewProcess(pid)
	if err != nil {
		if utils.ErrorIs(err, process.ErrorProcessNotRunning) {
			return nil
		}

		return fmt.Errorf("getting process for PID %d: %w", pid, err)
	}

	// Terminate the process.
	if err := proc.Terminate(); err != nil {
		return fmt.Errorf("terminating process: %w", err)
	}

	return nil
}

// PostDown performs cleanup operations after the client process is terminated.
func (c *Client) PostDown() error {
	// Removes configuration file.
	cfgFile := c.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("removing config file %q: %w", cfgFile, err)
	}

	// Remove PID file.
	pidFile := c.pidFilePath()
	if err := utils.RemoveFile(pidFile); err != nil {
		return fmt.Errorf("removing PID file %q: %w", pidFile, err)
	}

	return nil
}

// Statistics returns dummy statistics for now (to be implemented).
func (c *Client) Statistics(_ context.Context) (int64, int64, error) {
	return 0, 0, errors.New("not implemented")
}
