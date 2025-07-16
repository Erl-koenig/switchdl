package cmd

import (
	"fmt"
	"os"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
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

		if videoCfg.AccessToken == "" {
			token, err := keyring.Get(keyringconfig.Service, keyringconfig.User)
			if err == nil {
				videoCfg.AccessToken = token
			} else {
				return fmt.Errorf("access token not found. Run 'switchdl configure' or provide it with the --token flag")
			}
		}

		client := media.NewClient(videoCfg.AccessToken)

		if err := os.MkdirAll(videoCfg.OutputDir, 0755); err != nil {
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
