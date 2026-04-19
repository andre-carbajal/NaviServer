package cmd

import (
	"naviserver/internal/cli/ui"

	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open interactive TUI",
	Run: func(cmd *cobra.Command, args []string) {
		ui.RunMainTUI(Client)
	},
}

func init() {
	RootCmd.AddCommand(tuiCmd)
}
