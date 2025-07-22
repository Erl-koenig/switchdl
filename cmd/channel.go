package cmd

import (
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var channelCmd = &cobra.Command{
	Use:   "channel <id>",
	Short: "Download videos from one or multiple channels",
	Long: `Download videos from one or more SwitchTube channels by providing their unique channel IDs.
You can either download all videos at once or select which ones specifically.`,
	Example: ` switchdl channel abcdef1234
 switchdl channel abcdef1234 ghijk56789 -a`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := media.NewClient(downloadCfg.AccessToken)
		downloadCfg.All = viper.GetBool("all")
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
	cobra.CheckErr(viper.BindPFlag("all", channelCmd.Flags().Lookup("all")))
}
