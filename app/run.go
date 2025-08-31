package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
)

// Run initializes the root Cobra command, sets up signal handling,
// and executes the CLI with proper error and exit handling.
func Run(buildRootCmd func(userDir string) *cobra.Command) {
	// Retrieve the user's home directory.
	userDir, err := os.UserHomeDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: Unable to determine user home directory: %v\n", err)
		os.Exit(1)
	}

	// Enable Cobra's feature to traverse and execute hooks for commands.
	cobra.EnableTraverseRunHooks = true

	// Initialize the root command for the application.
	cmd := buildRootCmd(userDir)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// Create a context that listens for SIGINT and SIGTERM signals
	ctx, cancel := SignalContext(context.Background())
	defer cancel(nil)

	// Execute the root command.
	if err := cmd.ExecuteContext(ctx); err != nil {
		// If not already a RunError or ShutdownError, wrap as RunError.
		if !errors.As(err, new(*RunError)) && !errors.As(err, new(*ShutdownError)) {
			err = NewRunError(err)
		}

		// Extract signal error once.
		sigErr := new(SignalError)
		isSigErr := errors.As(context.Cause(ctx), &sigErr)
		isShutdownErr := errors.As(err, new(*ShutdownError))

		// Print error if:
		// - it wasn't a signal error (normal failure), OR
		// - it was a shutdown error triggered by signal.
		if !isSigErr || isShutdownErr {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		// Decide exit code inline.
		if isSigErr {
			if n, ok := sigErr.Signal.(syscall.Signal); ok {
				os.Exit(128 + int(n))
			}
		}

		os.Exit(1)
	}
}
