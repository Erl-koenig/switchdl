package cmd

import (
	"fmt"
	"os"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
)

var videoCfg media.DownloadConfig

var videoCmd = &cobra.Command{
	Use:   "video <id>",
	Short: "Download one or multiple videos specified by their id",
	Args:  cobra.MinimumNArgs(1),
	Example: ` switchdl video 1234567890
 switchdl video 1234567890 9876543210 3134859203 
 switchdl video 1234567890 -o /path/to/dir --filename custom_name.mp4 -w -s`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if videoCfg.Filename != "" && len(args) > 1 {
			return fmt.Errorf(
				"custom filename (-f/--filename) can only be used when downloading a single video",
			)
		}

		videoCfg.VideoIDs = args

		token, err := keyringconfig.GetAccessToken(videoCfg.AccessToken)
		if err != nil {
			return err
		}
		videoCfg.AccessToken = token

		client := media.NewClient(videoCfg.AccessToken)

		if err := os.MkdirAll(videoCfg.OutputDir, media.DefaultDirectoryPermissions); err != nil {
			return fmt.Errorf("error creating output directory: %w", err)
		}

		summary := client.DownloadVideos(cmd.Context(), &videoCfg)

		if summary.Succeeded == 0 {
			return fmt.Errorf("failed to download any videos")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(videoCmd)
	videoCmd.Flags().
		StringVarP(&videoCfg.OutputDir, "output-dir", "o", ".", "Output directory path")
	videoCmd.Flags().
		StringVarP(&videoCfg.Filename, "filename", "f", "", "Output filename (defaults to video title)")
	videoCmd.Flags().
		StringVarP(&videoCfg.AccessToken, "token", "t", "", "Access token for API authentication (overrides configured token)")
	videoCmd.Flags().
		BoolVarP(&videoCfg.Overwrite, "overwrite", "w", false, "Overwrite existing files")
	videoCmd.Flags().
		BoolVarP(&videoCfg.SelectVariant, "select-variant", "s", false, "List all video variants (quality) and prompt for selection")
}
