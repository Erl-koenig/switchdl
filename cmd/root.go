// Package cmd provides the commands and flags managed by cobra for the CLI
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Erl-koenig/switchdl/internal/keyringconfig"
	"github.com/Erl-koenig/switchdl/internal/media"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const configName = "config"

var downloadCfg media.DownloadConfig

var rootCmd = &cobra.Command{
	Use:   "switchdl",
	Short: "A CLI tool for downloading videos from SwitchTube",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "man" || cmd.Name() == "completion" || cmd.Name() == "version" {
			return nil
		}

		if err := viper.Unmarshal(&downloadCfg); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		if downloadCfg.Overwrite && downloadCfg.Skip {
			return errors.New("cannot use --overwrite (-w) and --skip (-s) flags together")
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
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().
		StringVarP(&downloadCfg.OutputDir, "output-dir", "o", ".", "Output directory path")
	rootCmd.PersistentFlags().BoolVarP(&downloadCfg.Skip, "skip", "s", false, "Skip existing files")
	rootCmd.PersistentFlags().
		BoolVarP(&downloadCfg.Overwrite, "overwrite", "w", false, "Force overwrite of existing files")
	rootCmd.PersistentFlags().
		BoolVarP(&downloadCfg.SelectVariant, "select-variant", "v", false, "List all video variants (quality) and prompt for selection")
	rootCmd.PersistentFlags().
		String("token", "", "Access token for API authentication (overrides configured token)")

	cobra.CheckErr(viper.BindPFlag("output-dir", rootCmd.PersistentFlags().Lookup("output-dir")))
	cobra.CheckErr(viper.BindPFlag("skip", rootCmd.PersistentFlags().Lookup("skip")))
	cobra.CheckErr(viper.BindPFlag("overwrite", rootCmd.PersistentFlags().Lookup("overwrite")))
	cobra.CheckErr(
		viper.BindPFlag("select-variant", rootCmd.PersistentFlags().Lookup("select-variant")),
	)
}

func initConfig() {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)

	configPath := filepath.Join(home, ".config", "switchdl") // ~/.config/switchdl
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".") // cwd
	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("SWITCHDL")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if err = viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
