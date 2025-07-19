package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Manage your SwitchTube access token",
	Long: `The configure command allows you to set, show, validate, or delete your SwitchTube access token.

To set or update your token:
  switchdl configure

To check if a token is currently stored:
  switchdl configure show

To validate the stored token against the API:
  switchdl configure validate

To delete the stored token:
  switchdl configure delete`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your SwitchTube access token: ")
		token, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}
		token = strings.TrimSpace(token)
		if err := keyringconfig.SetAccessToken(token); err != nil {
			return err
		}
		fmt.Println("Access token successfully saved.")

		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Check if an access token is currently stored",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := keyringconfig.GetAccessToken("")
		switch err {
		case nil:
			fmt.Println("An access token is currently stored.")
		case keyring.ErrNotFound:
			fmt.Println("No access token is currently stored.")
		default:
			return fmt.Errorf("failed to check token status: %w", err)
		}
		return nil
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the stored access token with the SwitchTube API",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := keyringconfig.GetAccessToken("")
		if err != nil {
			return err
		}

		client := media.NewClient(token)
		if err := client.ValidateToken(cmd.Context()); err != nil {
			fmt.Println(err)
			if strings.Contains(err.Error(), "invalid or expired") {
				fmt.Println("Please run 'switchdl configure' to update it.")
			}
			return nil
		}
		fmt.Println("Access token is valid.")
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the stored access token",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := keyringconfig.DeleteAccessToken(); err != nil {
			return err
		}
		fmt.Println("Access token successfully deleted or was not found.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.AddCommand(showCmd)
	configureCmd.AddCommand(validateCmd)
	configureCmd.AddCommand(deleteCmd)
}
