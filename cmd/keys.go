package cmd

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"

	"github.com/sentinel-official/sentinel-go-sdk/config"
	"github.com/sentinel-official/sentinel-go-sdk/core"
	"github.com/sentinel-official/sentinel-go-sdk/core/input"
	"github.com/sentinel-official/sentinel-go-sdk/utils"
)

// NewKeysCmd creates and returns a new Cobra command for key management sub-commands.
func NewKeysCmd(cfg *config.KeyringConfig) *cobra.Command {
	// Initialize a base client
	c := core.NewBaseClient()

	cmd := &cobra.Command{
		Use:          "keys",
		Short:        "Sub-commands for managing keys",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Validate the provided configuration
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("failed to validate config: %w", err)
			}

			// Setup the keyring for the base client
			if err := c.SetupKeyring(cfg); err != nil {
				return fmt.Errorf("failed to setup keyring: %w", err)
			}

			return nil
		},
	}

	// Add sub-commands for key management
	cmd.AddCommand(
		keysAddCmd(c),
		keysDeleteCmd(c),
		keysListCmd(c),
		keysShowCmd(c),
	)

	// Configure persistent flags for the command
	cfg.SetForFlags(cmd.PersistentFlags())

	return cmd
}

// keysAddCmd creates a new key with the specified name, mnemonic, and bip39 passphrase.
func keysAddCmd(c *core.Client) *cobra.Command {
	// Declare variables for flags
	var account uint32
	var coinType uint32
	var index uint32
	var outputFormat = "text"

	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new key with the specified name and optional mnemonic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if the key already exists
			if _, err := c.Key(args[0]); err == nil {
				return fmt.Errorf("key with name '%s' already exists", args[0])
			}

			// Initialize a reader for user input
			reader := bufio.NewReader(cmd.InOrStdin())

			// Prompt for mnemonic
			mnemonic, err := input.GetString("Enter your bip39 mnemonic, or hit enter to generate one:\n", reader)
			if err != nil {
				return fmt.Errorf("failed to get input: %w", err)
			}

			// Validate the provided mnemonic
			if mnemonic != "" && !bip39.IsMnemonicValid(mnemonic) {
				return errors.New("invalid mnemonic")
			}

			// Prompt for bip39 passphrase
			bip39Pass, err := input.GetPassword("Enter your bip39 passphrase, or hit enter to use the default:", reader)
			if err != nil {
				return fmt.Errorf("failed to get input: %w", err)
			}

			// Confirm passphrase if provided
			if bip39Pass != "" {
				confirmPass, err := input.GetPassword("Confirm bip39 passphrase:", reader)
				if err != nil {
					return fmt.Errorf("failed to get input: %w", err)
				}

				if bip39Pass != confirmPass {
					return errors.New("bip39 passphrase does not match")
				}
			}

			// Create the key with the provided details
			newMnemonic, key, err := c.CreateKey(args[0], mnemonic, bip39Pass, coinType, account, index)
			if err != nil {
				return fmt.Errorf("failed to create new key: %w", err)
			}

			// Format the output for the created key
			output, err := keyring.MkAccKeyOutput(key)
			if err != nil {
				return fmt.Errorf("failed to create key output: %w", err)
			}

			// Display a mnemonic warning if a new mnemonic is generated
			if newMnemonic != mnemonic {
				cmd.Printf("")
				cmd.Printf("####################################################################")
				cmd.Printf("WARNING: YOU MUST SAVE THE FOLLOWING MNEMONIC SECURELY!")
				cmd.Printf("THIS MNEMONIC IS REQUIRED TO RECOVER YOUR KEY.")
				cmd.Printf("IF YOU LOSE THIS MNEMONIC, YOU WILL NOT BE ABLE TO RECOVER YOUR KEY.")
				cmd.Printf("####################################################################")
				cmd.Printf("")

				output.Mnemonic = newMnemonic
			}

			// Output the key details
			if err := utils.Writeln(cmd.OutOrStdout(), output, outputFormat); err != nil {
				return fmt.Errorf("failed to write to output: %w", err)
			}

			cmd.Println("Key created successfully")
			return nil
		},
	}

	// Bind flags to variables
	cmd.Flags().Uint32Var(&account, "account", account, "account number to use for key creation")
	cmd.Flags().Uint32Var(&coinType, "coin-type", coinType, "coin type to use for key creation")
	cmd.Flags().Uint32Var(&index, "index", index, "index to use for key creation")
	cmd.Flags().StringVar(&outputFormat, "output-format", outputFormat, "format for command output (json or text)")

	return cmd
}

// keysDeleteCmd removes the key with the specified name.
func keysDeleteCmd(c *core.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete the key with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if the key exists
			if _, err := c.Key(args[0]); err != nil {
				return fmt.Errorf("failed to retreive key: %w", err)
			}

			// Initialize a reader for user input
			reader := bufio.NewReader(cmd.InOrStdin())

			// Prompt for confirmation before deletion
			confirm, err := input.GetConfirmation("Are you sure you want to delete this key? [y/N]:", reader)
			if err != nil {
				return fmt.Errorf("failed to get input: %w", err)
			}
			if !confirm {
				return errors.New("deletion aborted")
			}

			// Delete the key
			if err := c.DeleteKey(args[0]); err != nil {
				return fmt.Errorf("failed to delete key: %w", err)
			}

			cmd.Println("Key deleted successfully")
			return nil
		},
	}

	return cmd
}

// keysListCmd lists all the available keys.
func keysListCmd(c *core.Client) *cobra.Command {
	// Declare variables for flags
	var outputFormat = "text"

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Fetch the list of keys from the client
			keys, err := c.Keys()
			if err != nil {
				return fmt.Errorf("failed to retreive keys: %w", err)
			}

			// Format the keys for output
			output, err := keyring.MkAccKeysOutput(keys)
			if err != nil {
				return fmt.Errorf("failed to create keys output: %w", err)
			}

			// Output the keys in the specified format
			if err := utils.Writeln(cmd.OutOrStdout(), output, outputFormat); err != nil {
				return fmt.Errorf("failed to write to output: %w", err)
			}

			return nil
		},
	}

	// Bind flags to variables
	cmd.Flags().StringVar(&outputFormat, "output-format", outputFormat, "format for command output (json or text)")

	return cmd
}

// keysShowCmd displays details of the key with the specified name.
func keysShowCmd(c *core.Client) *cobra.Command {
	// Declare variables for flags
	var outputFormat = "text"

	cmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show details of the key with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Retrieve key details from the client
			key, err := c.Key(args[0])
			if err != nil {
				return fmt.Errorf("failed to retreive key: %w", err)
			}

			// Format the key for output
			output, err := keyring.MkAccKeyOutput(key)
			if err != nil {
				return fmt.Errorf("failed to create key output: %w", err)
			}

			// Output the key details in the specified format
			if err := utils.Writeln(cmd.OutOrStdout(), output, outputFormat); err != nil {
				return fmt.Errorf("failed to write to output: %w", err)
			}

			return nil
		},
	}

	// Bind flags to variables
	cmd.Flags().StringVar(&outputFormat, "output-format", outputFormat, "format for command output (json or text)")

	return cmd
}
