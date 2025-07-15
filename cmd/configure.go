package cmd

import (
	"bufio"
	"fmt"
	"net/http"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 { // default to setting the token if no subcommand provided
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter your SwitchTube access token: ")
			token, _ := reader.ReadString('\n')
			token = strings.TrimSpace(token)

			if token == "" {
				return fmt.Errorf("access token cannot be empty")
			}

			err := keyring.Set(keyringconfig.Service, keyringconfig.User, token)
			if err != nil {
				return fmt.Errorf("failed to save token to keyring: %w", err)
			}

			fmt.Println("Access token successfully saved.")
			return nil
		}
		return cmd.Help()
	},
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Check if an access token is currently stored",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := keyring.Get(keyringconfig.Service, keyringconfig.User)
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
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := keyring.Get(keyringconfig.Service, keyringconfig.User)
		if err != nil {
			if err == keyring.ErrNotFound {
				return fmt.Errorf(
					"no access token found. Please run 'switchdl configure' to set one")
			}
			return fmt.Errorf("failed to retrieve token from keyring: %w", err)
		}

		client := &http.Client{}
		req, err := http.NewRequest("GET", media.SwitchTubeBaseURL+"/api/v1/profiles/me", nil)
		if err != nil {
			return fmt.Errorf("failed to create validation request: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Token %s", token))
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send validation request: %w", err)
		}
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("failed to close response body: %w", cerr)
			}
		}()

		switch resp.StatusCode {
		case http.StatusOK:
			fmt.Println("Access token is valid.")
		case http.StatusUnauthorized, http.StatusForbidden:
			fmt.Printf(
				"Access token is invalid or expired (HTTP %d). Please run 'switchdl configure' to update it.\n",
				resp.StatusCode,
			)
		default:
			return fmt.Errorf(
				"unexpected API response for token validation: HTTP %d",
				resp.StatusCode,
			)
		}
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the stored access token",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := keyring.Delete(keyringconfig.Service, keyringconfig.User)
		switch err {
		case nil:
			fmt.Println("Access token successfully deleted.")
		case keyring.ErrNotFound:
			fmt.Println("No access token was found to delete.")
		default:
			return fmt.Errorf("failed to delete token: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
	configureCmd.AddCommand(showCmd)
	configureCmd.AddCommand(validateCmd)
	configureCmd.AddCommand(deleteCmd)
}
