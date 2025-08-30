package app

import (
	"context"
	"errors"
	"fmt"
	"os"

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
		// If context was canceled by a signal
		if errors.As(context.Cause(ctx), new(*SignalError)) {
			// StartError or WaitError during signal shutdown, treat as clean exit
			if errors.As(err, new(*StartError)) || errors.As(err, new(*WaitError)) {
				os.Exit(0)
			}
		}

		// Otherwise, print the error and exit with non-zero code.
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
