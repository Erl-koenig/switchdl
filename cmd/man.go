package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages for switchdl",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return doc.GenManTree(rootCmd, &doc.GenManHeader{
			Title:   "SWITCHDL",
			Section: "1",
		}, "man/")
	},
}

func init() {
	rootCmd.AddCommand(manCmd)
}
