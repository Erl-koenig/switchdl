package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of switchdl",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("switchdl %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
