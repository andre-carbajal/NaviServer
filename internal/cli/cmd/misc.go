package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var loadersCmd = &cobra.Command{
	Use:   "loaders",
	Short: "List available loaders",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleListLoaders()
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleCheckUpdates()
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the daemon",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleRestartDaemon()
	},
}

func init() {
	RootCmd.AddCommand(loadersCmd, updateCmd, restartCmd)
}

func handleListLoaders() error {
	loaders, err := Client.ListLoaders()
	if err != nil {
		return fmt.Errorf("list loaders: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":  "list_loaders",
			"status":  "ok",
			"loaders": loaders,
		})
	}

	rows := make([][]string, 0, len(loaders))
	for _, l := range loaders {
		rows = append(rows, []string{l})
	}
	printTable([]string{"LOADER"}, rows)
	return nil
}

func handleCheckUpdates() error {
	info, err := Client.CheckUpdates()
	if err != nil {
		return fmt.Errorf("check updates: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "check_updates",
			"status": "ok",
			"update": info,
		})
	}

	availability := "NO"
	if info.UpdateAvailable {
		availability = "YES"
	}
	printTable([]string{"KEY", "VALUE"}, [][]string{
		{"CURRENT_VERSION", info.CurrentVersion},
		{"LATEST_VERSION", info.LatestVersion},
		{"UPDATE_AVAILABLE", availability},
		{"RELEASE_URL", info.ReleaseURL},
	})

	return nil
}

func handleRestartDaemon() error {
	if err := Client.RestartDaemon(); err != nil {
		return fmt.Errorf("restart daemon: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]string{
			"action": "restart_daemon",
			"status": "ok",
		})
	}

	fmt.Println("OK  daemon restart command sent")
	return nil
}
