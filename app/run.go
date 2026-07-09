package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// Run initializes the root Cobra command, sets up signal handling,
// and executes the CLI with proper error and exit handling.
// It returns an exit code that should be used to terminate the program.
func Run(ctx context.Context, buildRootCmd func(userDir string) *cobra.Command) (exitCode int) {
	// Retrieve the user's home directory.
	userDir, err := os.UserHomeDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: Unable to determine user home directory: %v\n", err)

		return 1
	}

	// Enable Cobra's feature to traverse and execute hooks for commands.
	cobra.EnableTraverseRunHooks = true

	// Initialize the root command for the application.
	cmd := buildRootCmd(userDir)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	// Create a context that listens for SIGINT and SIGTERM signals
	ctx, cancel := SignalContext(ctx)
	defer cancel(nil)

	// Execute the root command.
	if err := cmd.ExecuteContext(ctx); err != nil {
		isRunErr := utils.ErrorIs(err, ErrRun)
		isStopErr := utils.ErrorIs(err, ErrStop)

		// If not already a RunError or StopError, wrap as RunError.
		if !isRunErr && !isStopErr {
			err = NewErrRun(err)
		}

		// Extract signal error once.
		sigErr := new(SignalError)
		isSigErr := errors.As(context.Cause(ctx), &sigErr)

		// Print error if:
		// - it wasn't a signal error (normal failure), OR
		// - it was a stop error triggered by signal.
		if !isSigErr || isStopErr {
			_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		return sigErr.ExitCode()
	}

	return 0
}
