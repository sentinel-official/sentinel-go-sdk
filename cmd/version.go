package cmd

import (
	"github.com/spf13/cobra"

	"github.com/sentinel-official/sentinel-go-sdk/v2/version"
)

// NewVersionCmd creates and returns the version command.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the application version",
		Long: `Prints the current version of the application, including the Git commit and tag
used to build it. Information is provided by the sentinel-go-sdk/version package.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Print the version information from the sdk.
			cmd.Println(version.Get())
		},
	}

	return cmd
}
