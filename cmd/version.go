package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of your CLI tool",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("switchdl %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
