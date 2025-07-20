package cmd

import (
	"fmt"
	"os"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
)

var videoCmd = &cobra.Command{
	Use:   "video <id>",
	Short: "Download one or more videos specified by their id",
	Example: ` switchdl video 1234567890
 switchdl video 1234567890 9876543210 3134859203 
 switchdl video 1234567890 -o /path/to/dir --filename custom_name.mp4 -w -v`,
	Args: cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if downloadCfg.Overwrite && downloadCfg.Skip {
			return fmt.Errorf("cannot use --overwrite (-w) and --skip (-s) flags together")
		}

		filename, _ := cmd.Flags().GetString("filename")
		if filename != "" && len(args) > 1 {
			return fmt.Errorf(
				"custom filename (-f/--filename) can only be used when downloading a single video",
			)
		}

		token, err := keyringconfig.GetAccessToken(downloadCfg.AccessToken)
		if err != nil {
			return err
		}
		downloadCfg.AccessToken = token

		return os.MkdirAll(downloadCfg.OutputDir, media.DefaultDirectoryPermissions)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Video-specific flags
		downloadCfg.VideoIDs = args
		downloadCfg.Filename, _ = cmd.Flags().GetString("filename")

		client := media.NewClient(downloadCfg.AccessToken)
		summary := client.DownloadVideos(cmd.Context(), &downloadCfg)

		if summary.Succeeded == 0 {
			return fmt.Errorf("failed to download any videos")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.Flags().StringP("filename", "f", "", "Output filename (defaults to video title)")
}
