package cmd

import (
	"naviserver/internal/cli/ui"

	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open interactive TUI",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.RunMainTUI(Client)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(tuiCmd)
}
