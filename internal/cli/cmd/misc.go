package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "Manage port range",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var portsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get port range",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleGetPortRange()
	},
}

var portsStart, portsEnd int
var portsSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set port range",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if portsStart == 0 || portsEnd == 0 {
			return newValidationError("you must specify both --start and --end")
		}
		return handleSetPortRange(portsStart, portsEnd)
	},
}

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
	portsSetCmd.Flags().IntVar(&portsStart, "start", 0, "Start port")
	portsSetCmd.Flags().IntVar(&portsEnd, "end", 0, "End port")
	portsCmd.AddCommand(portsGetCmd, portsSetCmd)

	RootCmd.AddCommand(portsCmd, loadersCmd, updateCmd, restartCmd)
}

func handleGetPortRange() error {
	pr, err := Client.GetPortRange()
	if err != nil {
		return fmt.Errorf("get port range: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "get_port_range",
			"status": "ok",
			"port_range": map[string]int{
				"start": pr.Start,
				"end":   pr.End,
				"range": pr.End - pr.Start + 1,
			},
		})
	}

	printTable([]string{"KEY", "VALUE"}, [][]string{
		{"START_PORT", fmt.Sprintf("%d", pr.Start)},
		{"END_PORT", fmt.Sprintf("%d", pr.End)},
		{"RANGE", fmt.Sprintf("%d", pr.End-pr.Start+1)},
	})
	return nil
}

func handleSetPortRange(start, end int) error {
	if err := Client.SetPortRange(start, end); err != nil {
		return fmt.Errorf("set port range: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "set_port_range",
			"status": "ok",
			"start":  start,
			"end":    end,
		})
	}

	fmt.Println("OK  port configuration updated")
	fmt.Printf("New range: %d - %d\n", start, end)
	return nil
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
