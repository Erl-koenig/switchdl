package cmd

import (
	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
)

var channelCfg media.DownloadConfig

var channelCmd = &cobra.Command{
	Use:   "channel <id>",
	Short: "Download videos from one or multiple channels",
	Long: `Download videos from one or more SwitchTube channels by providing their unique channel IDs.
You can either download all videos at once or select which ones specifically.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, channelID := range args {
			channelCfg.ChannelID = channelID

			token, err := keyringconfig.GetAccessToken(channelCfg.AccessToken)
			if err != nil {
				return err
			}
			channelCfg.AccessToken = token

			client := media.NewClient(channelCfg.AccessToken)

			if err := client.DownloadChannel(cmd.Context(), &channelCfg); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(channelCmd)
	channelCmd.Flags().
		StringVarP(&channelCfg.OutputDir, "output-dir", "o", ".", "Output directory path")
	channelCmd.Flags().
		StringVarP(&channelCfg.AccessToken, "token", "t", "", "Access token for API authentication (overrides configured token)")
	channelCmd.Flags().
		BoolVarP(&channelCfg.Overwrite, "overwrite", "w", false, "Force overwrite of existing files")
	channelCmd.Flags().
		BoolVarP(&channelCfg.All, "all", "a", false, "Download all videos without prompting")
	channelCmd.Flags().
		BoolVarP(&channelCfg.SelectVariant, "select-variant", "s", false, "List all video variants (quality) and prompt for selection")
}
