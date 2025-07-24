package cmd

import (
	"errors"

	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var videoCmd = &cobra.Command{
	Use:   "video <id>",
	Short: "Download one or more videos specified by their id",
	Example: `  switchdl video 1234567890
  switchdl video 1234567890 9876543210 3134859203
  switchdl video 1234567890 -o /path/to/dir -f custom_name.mp4 -w -v`,
	Args: cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		filename := viper.GetString("filename")
		if filename != "" && len(args) > 1 {
			return errors.New(
				"custom filename (-f/--filename) can only be used when downloading a single video",
			)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		downloadCfg.VideoIDs = args
		downloadCfg.Filename = viper.GetString("filename")

		client := media.NewClient(downloadCfg.AccessToken)
		summary := client.DownloadVideos(cmd.Context(), &downloadCfg)

		if summary.Succeeded == 0 {
			return errors.New("failed to download any videos")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.Flags().StringP("filename", "f", "", "Output filename (defaults to video title)")
	cobra.CheckErr(viper.BindPFlag("filename", videoCmd.Flags().Lookup("filename")))
}
