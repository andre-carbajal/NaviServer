package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage daemon settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var settingsPortRangeCmd = &cobra.Command{
	Use:   "port-range",
	Short: "Manage daemon port range",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var settingsPortRangeGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get configured port range",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleSettingsPortRangeGet()
	},
}

var (
	settingsPortRangeStart int
	settingsPortRangeEnd   int
)

var settingsPortRangeSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set configured port range",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("start") || !cmd.Flags().Changed("end") {
			return newValidationError("you must specify both --start and --end")
		}
		if settingsPortRangeStart <= 0 || settingsPortRangeEnd <= 0 {
			return newValidationError("--start and --end must be greater than 0")
		}
		if settingsPortRangeStart > settingsPortRangeEnd {
			return newValidationError("--start cannot be greater than --end")
		}
		return handleSettingsPortRangeSet(settingsPortRangeStart, settingsPortRangeEnd)
	},
}

var settingsPublicIPCmd = &cobra.Command{
	Use:   "public-ip",
	Short: "Manage public IP/address",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var settingsPublicIPGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get configured public IP/address",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleSettingsPublicIPGet()
	},
}

var settingsPublicIPValue string

var settingsPublicIPSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set public IP/address",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		value := strings.TrimSpace(settingsPublicIPValue)
		if value == "" {
			return newValidationError("you must specify --value")
		}
		return handleSettingsPublicIPSet(value)
	},
}

var settingsInterfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "List detected network interfaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var settingsInterfacesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List detected network interface IPv4 addresses",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleSettingsInterfacesList()
	},
}

var settingsLogBufferCmd = &cobra.Command{
	Use:   "log-buffer",
	Short: "Manage console log buffer size",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var settingsLogBufferGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get console log buffer size",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleSettingsLogBufferGet()
	},
}

var settingsLogBufferLines int

var settingsLogBufferSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set console log buffer size",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("lines") {
			return newValidationError("you must specify --lines")
		}
		if settingsLogBufferLines < 0 {
			return newValidationError("--lines must be greater than or equal to 0")
		}
		return handleSettingsLogBufferSet(settingsLogBufferLines)
	},
}

func init() {
	settingsPortRangeSetCmd.Flags().IntVar(&settingsPortRangeStart, "start", 0, "Start port")
	settingsPortRangeSetCmd.Flags().IntVar(&settingsPortRangeEnd, "end", 0, "End port")
	settingsPublicIPSetCmd.Flags().StringVar(&settingsPublicIPValue, "value", "", "Public IP or hostname")
	settingsLogBufferSetCmd.Flags().IntVar(&settingsLogBufferLines, "lines", 0, "Number of log lines to retain")

	settingsPortRangeCmd.AddCommand(settingsPortRangeGetCmd, settingsPortRangeSetCmd)
	settingsPublicIPCmd.AddCommand(settingsPublicIPGetCmd, settingsPublicIPSetCmd)
	settingsInterfacesCmd.AddCommand(settingsInterfacesListCmd)
	settingsLogBufferCmd.AddCommand(settingsLogBufferGetCmd, settingsLogBufferSetCmd)

	settingsCmd.AddCommand(
		settingsPortRangeCmd,
		settingsPublicIPCmd,
		settingsInterfacesCmd,
		settingsLogBufferCmd,
	)

	RootCmd.AddCommand(settingsCmd)
}

func handleSettingsPortRangeGet() error {
	pr, err := Client.GetPortRange()
	if err != nil {
		return fmt.Errorf("get settings port range: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "get_settings_port_range",
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

func handleSettingsPortRangeSet(start, end int) error {
	if err := Client.SetPortRange(start, end); err != nil {
		return fmt.Errorf("set settings port range: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "set_settings_port_range",
			"status": "ok",
			"start":  start,
			"end":    end,
			"range":  end - start + 1,
		})
	}

	fmt.Println("OK  settings port range updated")
	fmt.Printf("New range: %d - %d\n", start, end)
	return nil
}

func handleSettingsPublicIPGet() error {
	publicIP, err := Client.GetPublicIP()
	if err != nil {
		return fmt.Errorf("get settings public ip: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":    "get_settings_public_ip",
			"status":    "ok",
			"public_ip": publicIP.PublicIP,
		})
	}

	printTable([]string{"KEY", "VALUE"}, [][]string{{"PUBLIC_IP", publicIP.PublicIP}})
	return nil
}

func handleSettingsPublicIPSet(value string) error {
	if err := Client.SetPublicIP(value); err != nil {
		return fmt.Errorf("set settings public ip: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":    "set_settings_public_ip",
			"status":    "ok",
			"public_ip": value,
		})
	}

	fmt.Printf("OK  public IP updated: %s\n", value)
	return nil
}

func handleSettingsInterfacesList() error {
	interfaces, err := Client.GetNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("list settings interfaces: %w", err)
	}

	items := uniqueSortedValues(interfaces.Interfaces)

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":     "list_settings_interfaces",
			"status":     "ok",
			"interfaces": items,
		})
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item})
	}
	printTable([]string{"INTERFACE_IP"}, rows)
	return nil
}

func handleSettingsLogBufferGet() error {
	logBuffer, err := Client.GetLogBufferSize()
	if err != nil {
		return fmt.Errorf("get settings log buffer: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "get_settings_log_buffer",
			"status": "ok",
			"log_buffer": map[string]int{
				"lines": logBuffer.LogBufferSize,
			},
		})
	}

	printTable([]string{"KEY", "VALUE"}, [][]string{{"LINES", fmt.Sprintf("%d", logBuffer.LogBufferSize)}})
	return nil
}

func handleSettingsLogBufferSet(lines int) error {
	if err := Client.SetLogBufferSize(lines); err != nil {
		return fmt.Errorf("set settings log buffer: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "set_settings_log_buffer",
			"status": "ok",
			"lines":  lines,
		})
	}

	fmt.Printf("OK  log buffer updated: %d lines\n", lines)
	return nil
}

func uniqueSortedValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}
