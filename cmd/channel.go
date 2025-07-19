package cmd

import (
	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
)

var channelCfg media.DownloadConfig

var channelCmd = &cobra.Command{
	Use:   "channel <id>",
	Short: "Download videos from a channel",
	Long: `Download videos from a SwitchTube channel.
You can either download all videos at once or select which ones specifically.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		channelCfg.ChannelID = args[0]

		token, err := keyringconfig.GetAccessToken(channelCfg.AccessToken)
		if err != nil {
			return err
		}
		channelCfg.AccessToken = token

		client := media.NewClient(channelCfg.AccessToken)

		return client.DownloadChannel(cmd.Context(), &channelCfg)
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
}
