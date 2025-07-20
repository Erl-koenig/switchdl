package cmd

import (
	"fmt"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel <id>",
	Short: "Download videos from one or multiple channels",
	Long: `Download videos from one or more SwitchTube channels by providing their unique channel IDs.
You can either download all videos at once or select which ones specifically.`,
	Args: cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if downloadCfg.Overwrite && downloadCfg.Skip {
			return fmt.Errorf("cannot use --overwrite (-w) and --skip (-s) flags together")
		}

		token, err := keyringconfig.GetAccessToken(downloadCfg.AccessToken)
		if err != nil {
			return err
		}
		downloadCfg.AccessToken = token
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		client := media.NewClient(downloadCfg.AccessToken)

		// Channel-specific flags
		downloadCfg.All, _ = cmd.Flags().GetBool("all")

		for _, channelID := range args {
			downloadCfg.ChannelID = channelID
			if err := client.DownloadChannel(cmd.Context(), &downloadCfg); err != nil {
				return err // Return on the first channel that fails
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(channelCmd)
	channelCmd.Flags().BoolP("all", "a", false, "Download all videos without prompting")
}
