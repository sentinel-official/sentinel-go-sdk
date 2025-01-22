package cmd

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sentinel-official/sentinel-go-sdk/client"
	"github.com/sentinel-official/sentinel-go-sdk/client/input"
	"github.com/sentinel-official/sentinel-go-sdk/types"
)

// KeysCmd returns a new Cobra command for key management sub-commands.
func KeysCmd() *cobra.Command {
	c := client.NewBaseClient()
	rootCmd := &cobra.Command{
		Use:          "keys",
		Short:        "Sub-commands for managing keys",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Retrieve keyring configuration from environment variables or flags
			homeDir := viper.GetString("home")
			appName := viper.GetString("keyring.name")
			backend := viper.GetString("keyring.backend")

			// Initialize the protocol codec
			protoCodec := types.NewProtoCodec()

			// Create a new keyring instance
			kr, err := keyring.New(appName, backend, homeDir, cmd.InOrStdin(), protoCodec)
			if err != nil {
				return err
			}

			// Create a new client with the keyring and set it in the command context
			c.WithKeyring(kr)
			c.WithProtoCodec(protoCodec)

			return nil
		},
	}

	// Add sub-commands for key management
	rootCmd.AddCommand(
		keysAddCmd(c),
		keysDeleteCmd(c),
		keysListCmd(c),
		keysShowCmd(c),
	)

	// Add persistent flags
	rootCmd.PersistentFlags().String("keyring.backend", "os", "backend type for the keyring (e.g., 'os', 'file', or 'test')")
	rootCmd.PersistentFlags().String("keyring.name", "sentinel", "name identifier for the keyring")

	return rootCmd
}

// keysAddCmd creates a new key with the specified name, mnemonic, and bip39 passphrase.
func keysAddCmd(c *client.BaseClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new key with the specified name and optional mnemonic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			account := viper.GetUint32("key.account")
			coinType := viper.GetUint32("key.coin-type")
			index := viper.GetUint32("key.index")
			outputFormat := viper.GetString("output-format")

			// Check if the key already exists
			if _, err := c.Key(args[0]); err == nil {
				return fmt.Errorf("key with name '%s' already exists", args[0])
			}

			reader := bufio.NewReader(cmd.InOrStdin())

			// Prompt for mnemonic
			mnemonic, err := input.GetString("Enter your bip39 mnemonic, or hit enter to generate one:\n", reader)
			if err != nil {
				return err
			}

			if mnemonic != "" && !bip39.IsMnemonicValid(mnemonic) {
				return errors.New("invalid mnemonic")
			}

			// Prompt for bip39 passphrase
			bip39Pass, err := input.GetPassword("Enter your bip39 passphrase, or hit enter to use the default:", reader)
			if err != nil {
				return err
			}

			// Confirm passphrase if provided
			if bip39Pass != "" {
				confirmPass, err := input.GetPassword("Confirm bip39 passphrase:", reader)
				if err != nil {
					return err
				}

				if bip39Pass != confirmPass {
					return errors.New("bip39 passphrase does not match")
				}
			}

			// Create the key
			newMnemonic, key, err := c.CreateKey(args[0], mnemonic, bip39Pass, coinType, account, index)
			if err != nil {
				return err
			}

			output, err := keyring.MkAccKeyOutput(key)
			if err != nil {
				return err
			}

			if newMnemonic != mnemonic {
				writeMnemonicWarningToCmd(cmd)
				output.Mnemonic = newMnemonic
			}

			// Output the key information
			if err := writeOutputToCmd(cmd, output, outputFormat); err != nil {
				return err
			}

			cmd.Println("Key created successfully")
			return nil
		},
	}

	cmd.Flags().Uint32("key.account", 0, "account number to use for key creation")
	cmd.Flags().Uint32("key.coin-type", 0, "coin type to use for key creation")
	cmd.Flags().Uint32("key.index", 0, "index to use for key creation")
	cmd.Flags().String("output-format", "text", "format for command output (json or text)")

	return cmd
}

// keysDeleteCmd removes the key with the specified name.
func keysDeleteCmd(c *client.BaseClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete the key with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := c.Key(args[0]); err != nil {
				return err
			}

			reader := bufio.NewReader(cmd.InOrStdin())

			confirm, err := input.GetConfirmation("Are you sure you want to delete this key? [y/N]:", reader)
			if err != nil {
				return err
			}
			if !confirm {
				return errors.New("deletion aborted")
			}

			// Delete the key
			if err := c.DeleteKey(args[0]); err != nil {
				return err
			}

			cmd.Println("Key deleted successfully")
			return nil
		},
	}

	return cmd
}

// keysListCmd lists all the available keys.
func keysListCmd(c *client.BaseClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat := viper.GetString("output-format")

			// Fetch the list of keys
			keys, err := c.Keys()
			if err != nil {
				return err
			}

			output, err := keyring.MkAccKeysOutput(keys)
			if err != nil {
				return err
			}

			// Output the key list
			if err := writeOutputToCmd(cmd, output, outputFormat); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("output-format", "text", "format for command output (json or text)")

	return cmd
}

// keysShowCmd displays details of the key with the specified name.
func keysShowCmd(c *client.BaseClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show details of the key with the specified name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputFormat := viper.GetString("output-format")

			// Fetch the key details
			key, err := c.Key(args[0])
			if err != nil {
				return err
			}

			output, err := keyring.MkAccKeyOutput(key)
			if err != nil {
				return err
			}

			// Output the key details
			if err := writeOutputToCmd(cmd, output, outputFormat); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("output-format", "text", "format for command output (json or text)")

	return cmd
}
