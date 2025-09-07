package cmd

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
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
	c := core.NewClient()

	cmd := &cobra.Command{
		Use:          "keys",
		Short:        "Sub-commands for managing keys",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Setup the keyring for the base client
			if err := c.SetupKeyring(cfg); err != nil {
				return fmt.Errorf("setting up keyring: %w", err)
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
	hdPath := hd.CreateHDPath(118, 0, 0).String()
	outputFormat := "text"

	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new key with the specified name and optional mnemonic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if the key already exists
			ok, err := c.HasKey(args[0])
			if err != nil {
				return fmt.Errorf("checking if key %q exists: %w", args[0], err)
			}
			if ok {
				return fmt.Errorf("key %q already exists", args[0])
			}

			// Initialize a reader for user input
			reader := bufio.NewReader(cmd.InOrStdin())

			// Prompt for mnemonic
			mnemonic, err := input.GetString("Enter your BIP-39 mnemonic, or hit enter to generate one:\n", reader)
			if err != nil {
				return fmt.Errorf("getting BIP-39 mnemonic input: %w", err)
			}

			// Validate the provided mnemonic
			if mnemonic != "" && !bip39.IsMnemonicValid(mnemonic) {
				return errors.New("invalid BIP-39 mnemonic")
			}

			// Prompt for bip39 passphrase
			bip39Pass, err := input.GetPassword("Enter your BIP-39 passphrase, or hit enter to use the default:", reader)
			if err != nil {
				return fmt.Errorf("getting BIP-39 passphrase input: %w", err)
			}

			// Confirm passphrase if provided
			if bip39Pass != "" {
				confirmPass, err := input.GetPassword("Confirm BIP-39 passphrase:", reader)
				if err != nil {
					return fmt.Errorf("getting BIP-39 passphrase confirmation input: %w", err)
				}

				if bip39Pass != confirmPass {
					return errors.New("BIP-39 passphrase mismatch")
				}
			}

			// Create the key with the provided details
			newMnemonic, key, err := c.CreateKey(args[0], mnemonic, bip39Pass, hdPath)
			if err != nil {
				return fmt.Errorf("creating new key %q: %w", args[0], err)
			}

			// Format the output for the created key
			output, err := keyring.MkAccKeyOutput(key)
			if err != nil {
				return fmt.Errorf("preparing output for key %q: %w", key.Name, err)
			}

			// Display a mnemonic warning if a new mnemonic is generated
			if newMnemonic != mnemonic {
				cmd.Println("")
				cmd.Println("####################################################################")
				cmd.Println("WARNING: YOU MUST SAVE THE FOLLOWING MNEMONIC SECURELY!")
				cmd.Println("THIS MNEMONIC IS REQUIRED TO RECOVER YOUR KEY.")
				cmd.Println("IF YOU LOSE THIS MNEMONIC, YOU WILL NOT BE ABLE TO RECOVER YOUR KEY.")
				cmd.Println("####################################################################")
				cmd.Println("")

				output.Mnemonic = newMnemonic
			}

			// Output the key details
			if err := utils.Writeln(cmd.OutOrStdout(), output, outputFormat); err != nil {
				return fmt.Errorf("writing key %q output: %w", key.Name, err)
			}

			cmd.Println("Key created successfully")

			return nil
		},
	}

	// Bind flags to variables
	cmd.Flags().StringVar(&hdPath, "hd-path", hdPath, "full absolute hd path of the bip44 params")
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
			ok, err := c.HasKey(args[0])
			if err != nil {
				return fmt.Errorf("checking if key %q exists: %w", args[0], err)
			}
			if !ok {
				return fmt.Errorf("key %q does not exist", args[0])
			}

			// Initialize a reader for user input
			reader := bufio.NewReader(cmd.InOrStdin())

			// Prompt for confirmation before deletion
			confirm, err := input.GetConfirmation("Are you sure you want to delete this key? [y/N]:", reader)
			if err != nil {
				return fmt.Errorf("getting key delete confirmation input: %w", err)
			}
			if !confirm {
				return fmt.Errorf("key %q deletion aborted", args[0])
			}

			// Delete the key
			if err := c.DeleteKey(args[0]); err != nil {
				return fmt.Errorf("deleting key %q: %w", args[0], err)
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
	outputFormat := "text"

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Fetch the list of keys from the client
			keys, err := c.Keys()
			if err != nil {
				return fmt.Errorf("retreiving keys: %w", err)
			}

			// Format the keys for output
			output, err := keyring.MkAccKeysOutput(keys)
			if err != nil {
				return fmt.Errorf("preparing output for keys: %w", err)
			}

			// Output the keys in the specified format
			if err := utils.Writeln(cmd.OutOrStdout(), output, outputFormat); err != nil {
				return fmt.Errorf("writing keys to output: %w", err)
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
	outputFormat := "text"

	cmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show details of the key with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Retrieve key details from the client
			key, err := c.Key(args[0])
			if err != nil {
				return fmt.Errorf("retrieving key %q: %w", args[0], err)
			}

			// Format the key for output
			output, err := keyring.MkAccKeyOutput(key)
			if err != nil {
				return fmt.Errorf("preparing output for key %q: %w", key.Name, err)
			}

			// Output the key details in the specified format
			if err := utils.Writeln(cmd.OutOrStdout(), output, outputFormat); err != nil {
				return fmt.Errorf("writing key %q output: %w", key.Name, err)
			}

			return nil
		},
	}

	// Bind flags to variables
	cmd.Flags().StringVar(&outputFormat, "output-format", outputFormat, "format for command output (json or text)")

	return cmd
}
