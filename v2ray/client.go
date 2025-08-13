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
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/sentinel-official/sentinel-go-sdk/types"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Ensure Client implements the types.ClientService interface.
var _ types.ClientService = (*Client)(nil)

// Client represents a V2Ray client with associated command, home directory, and name.
type Client struct {
	cmd     *exec.Cmd // Command for running the V2Ray client.
	homeDir string    // Home directory for client files.
	name    string    // Name of the interface.

	cancel context.CancelFunc // Context cancel function to stop background tasks.
	ctx    context.Context    // Context for server lifecycle management.
	eg     *errgroup.Group    // Error group for managing background goroutines.
}

// NewClient creates a new Client instance.
func NewClient(appDir string) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(ctx)

	return &Client{
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
		return 0, fmt.Errorf("failed to check existence of pid file: %w", err)
	}
	if !exists {
		return 0, nil
	}

	// Read PID from the PID file.
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	// Convert PID data to integer.
	pid, err := strconv.ParseInt(string(data), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse pid: %w", err)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("invalid pid %d", pid)
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
		return fmt.Errorf("failed to write file: %w", err)
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
		return fmt.Errorf("failed to create home directory: %w", err)
	}

	// Construct the full path to the config file
	cfgFile := c.appConfigFilePath()

	// Check if the config file exists at the specified path
	cfgFileExists, err := utils.IsFileExists(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}

	// Write default config only if file doesn't exist or force flag is enabled
	if !cfgFileExists || force {
		cfg := DefaultClientConfig()
		if err := cfg.WriteAppConfig(cfgFile); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	return nil
}

// IsUp checks if the V2Ray client process is running.
func (c *Client) IsUp() (bool, error) {
	// Read PID from file.
	pid, err := c.readPIDFromFile()
	if err != nil {
		return false, fmt.Errorf("failed to read pid from file: %w", err)
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

		return false, fmt.Errorf("failed to get process: %w", err)
	}

	// Check if the process is running.
	ok, err := proc.IsRunning()
	if err != nil {
		return false, fmt.Errorf("failed to check running process: %w", err)
	}
	if !ok {
		return false, nil
	}

	// Retrieve the name of the process.
	name, err := proc.Name()
	if err != nil {
		return false, fmt.Errorf("failed to get process name: %w", err)
	}

	// Check if the process name matches constant v2ray.
	if name != v2ray {
		return false, nil
	}

	return true, nil
}

// PreUp writes the configuration to the config file before starting the client process.
func (c *Client) PreUp(req interface{}) error {
	// Set default client configuration
	cfg := DefaultClientConfig()

	// If a request is provided, attempt to cast it to a ClientConfig type
	if req != nil {
		if v, ok := req.(*ClientConfig); ok {
			cfg = v
		} else {
			return fmt.Errorf("invalid request type %T", req)
		}
	}

	// Initialize viper instance
	v := viper.New()

	// Construct the full path to the config file
	cfgFile := c.appConfigFilePath()

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
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	// Validate the unmarshalled config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to validate config file: %w", err)
	}

	// Write configuration to file.
	cfgFile = c.serviceConfigFilePath()
	if err := cfg.WriteServiceConfig(cfgFile); err != nil {
		return fmt.Errorf("failed to write config to file: %w", err)
	}

	return nil
}

// Up starts the V2Ray client process.
func (c *Client) Up(ctx context.Context) error {
	// Create a context that cancels if either server or input context is done.
	ctx, _ = utils.AnyDoneContext(c.ctx, ctx)

	// Constructs the command to start the V2Ray client.
	cfgFile := c.serviceConfigFilePath()
	c.cmd = exec.CommandContext(
		ctx,
		c.execFile(v2ray),
		strings.Fields(fmt.Sprintf("run --config %s", cfgFile))...,
	)

	// Starts the V2Ray client process.
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for the V2Ray process to finish in a separate goroutine.
	c.eg.Go(func() (err error) {
		if err = c.cmd.Wait(); err == nil {
			err = errors.New("exited unexpectedly")
		}

		// If context is canceled, we return nil immediately.
		if utils.ErrorIs(ctx.Err(), context.Canceled) {
			return nil
		}

		return fmt.Errorf("failed to wait for command: %w", err)
	})

	return nil
}

// PostUp performs operations after the client process is started.
func (c *Client) PostUp(_ context.Context) error {
	// Write PID to file.
	if err := c.writePIDToFile(c.cmd.Process.Pid); err != nil {
		return fmt.Errorf("failed to write pid to file: %w", err)
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
		return fmt.Errorf("failed to read pid from file: %w", err)
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

		return fmt.Errorf("failed to get process: %w", err)
	}

	// Terminate the process.
	if err := proc.Terminate(); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	return nil
}

// PostDown performs cleanup operations after the client process is terminated.
func (c *Client) PostDown() error {
	// Removes configuration file.
	cfgFile := c.serviceConfigFilePath()
	if err := utils.RemoveFile(cfgFile); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	// Removes PID file.
	pidFile := c.pidFilePath()
	if err := utils.RemoveFile(pidFile); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}

// Statistics returns dummy statistics for now (to be implemented).
func (c *Client) Statistics() (int64, int64, error) {
	return 0, 0, nil
}
