// Package cmd provides the commands and flags managed by cobra for the CLI
package cmd

import (
	"os"

	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
)

var downloadCfg media.DownloadConfig

var rootCmd = &cobra.Command{
	Use:   "switchdl",
	Short: "A CLI tool for downloading videos from SwitchTube",
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
