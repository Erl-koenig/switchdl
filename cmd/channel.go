package cmd

import (
	"fmt"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

type channelConfig struct {
	outputDir   string
	accessToken string
	channelID   string
	downloadAll bool
}

var channelCfg channelConfig

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Download videos from a channel",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf(
				`no channel ID provided. Usage: switchdl channel <ID>
				 Use 'switchdl --help' for more information`,
			)
		}

		channelCfg.channelID = args[0]

		if channelCfg.accessToken == "" {
			token, err := keyring.Get(keyringconfig.Service, keyringconfig.User)
			if err == nil {
				channelCfg.accessToken = token
			} else {
				return fmt.Errorf("access token not found. Run 'switchdl configure' or provide it with the --token flag")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(channelCmd)
	channelCmd.Flags().StringVarP(&channelCfg.outputDir, "output-dir", "o", ".", "Output directory path")
	channelCmd.Flags().StringVarP(&channelCfg.accessToken, "token", "t", "", "Access token for API authentication (overrides configured token)")
	channelCmd.Flags().BoolVarP(&channelCfg.downloadAll, "all", "a", false, "Download all videos from the channel")
}
