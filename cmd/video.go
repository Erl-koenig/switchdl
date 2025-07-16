package cmd

import (
	"fmt"
	"os"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
)

var videoCfg media.DownloadVideoConfig

var videoCmd = &cobra.Command{
	Use:   "video <video_id>",
	Short: "Download a video specified by its id",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf(
				`no video ID provided. Usage: switchdl video <ID>
				 Use 'switchdl --help' for more information`,
			)
		}
		videoCfg.VideoID = args[0]

		token, err := keyringconfig.GetAccessToken(videoCfg.AccessToken)
		if err != nil {
			return err
		}
		videoCfg.AccessToken = token

		client := media.NewClient(videoCfg.AccessToken)

		if err := os.MkdirAll(videoCfg.OutputDir, 0o755); err != nil {
			return fmt.Errorf("error creating output directory: %w", err)
		}

		if err := client.DownloadVideo(cmd.Context(), &videoCfg); err != nil {
			return fmt.Errorf("error downloading video: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.Flags().StringVarP(&videoCfg.OutputDir, "output-dir", "o", ".", "Output directory path")
	videoCmd.Flags().StringVarP(&videoCfg.Filename, "filename", "f", "", "Output filename (defaults to video title)")
	videoCmd.Flags().StringVarP(&videoCfg.AccessToken, "token", "t", "", "Access token for API authentication (overrides configured token)")
	videoCmd.Flags().BoolVarP(&videoCfg.Overwrite, "overwrite", "w", false, "Overwrite existing files")
}
