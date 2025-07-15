package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Download videos from a channel",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("channel called")
	},
}

func init() {
	rootCmd.AddCommand(channelCmd)
}
