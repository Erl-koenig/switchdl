package cmd

import (
	"fmt"
	"os"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

type downloadConfig struct {
	outputDir   string
	filename    string
	accessToken string
	videoID     string
}

var videoCfg downloadConfig

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
		videoCfg.videoID = args[0]

		if videoCfg.accessToken == "" {
			token, err := keyring.Get(keyringconfig.Service, keyringconfig.User)
			if err == nil {
				videoCfg.accessToken = token
			} else {
				return fmt.Errorf("access token not found. Run 'switchdl configure' or provide it with the --token flag")
			}
		}

		client := media.NewClient(videoCfg.accessToken)

		fmt.Println("Downloading video with ID:", videoCfg.videoID)

		if err := os.MkdirAll(videoCfg.outputDir, 0755); err != nil {
			return fmt.Errorf("error creating output directory: %w", err)
		}

		if err := client.DownloadVideo(cmd.Context(), videoCfg.videoID, videoCfg.outputDir, videoCfg.filename); err != nil {
			return fmt.Errorf("error downloading video: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.Flags().
		StringVarP(&videoCfg.outputDir, "output-dir", "o", ".", "Output directory path")
	videoCmd.Flags().
		StringVarP(&videoCfg.filename, "filename", "f", "", "Output filename (defaults to video title)")
	videoCmd.Flags().
		StringVarP(&videoCfg.accessToken, "token", "t", "", "Access token for API authentication (overrides configured token)")
}
