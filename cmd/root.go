// Package cmd provides the commands and flags managed by cobra for the CLI
package cmd

import (
	"fmt"
	"os"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
)

var downloadCfg media.DownloadConfig

var rootCmd = &cobra.Command{
	Use:   "switchdl",
	Short: "A CLI tool for downloading videos from SwitchTube",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "man" || cmd.Name() == "completion" || cmd.Name() == "version" {
			return nil
		}

		if downloadCfg.Overwrite && downloadCfg.Skip {
			return fmt.Errorf("cannot use --overwrite (-w) and --skip (-s) flags together")
		}

		token, err := keyringconfig.GetAccessToken(downloadCfg.AccessToken)
		if err != nil {
			return err
		}
		downloadCfg.AccessToken = token

		return os.MkdirAll(downloadCfg.OutputDir, media.DefaultDirectoryPermissions)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().
		StringVarP(&downloadCfg.OutputDir, "output-dir", "o", ".", "Output directory path")
	rootCmd.PersistentFlags().
		StringVarP(&downloadCfg.AccessToken, "token", "t", "", "Access token for API authentication (overrides configured token)")
	rootCmd.PersistentFlags().BoolVarP(&downloadCfg.Skip, "skip", "s", false, "Skip existing files")
	rootCmd.PersistentFlags().
		BoolVarP(&downloadCfg.Overwrite, "overwrite", "w", false, "Force overwrite of existing files")
	rootCmd.PersistentFlags().
		BoolVarP(&downloadCfg.SelectVariant, "select-variant", "v", false, "List all video variants (quality) and prompt for selection")
}
