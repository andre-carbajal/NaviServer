package cmd

import (
	"fmt"
	"naviserver/pkg/sdk"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var backupCreateCmd = &cobra.Command{
	Use:   "create [serverId] [name]",
	Short: "Create a backup",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 1 {
			name = args[1]
		}
		return handleBackupCreate(args[0], name)
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list [serverId]",
	Short: "List backups",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return handleListBackups(args[0])
		}
		return handleListAllBackups()
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleDeleteBackup(args[0])
	},
}

var restoreTarget, restoreName, restoreVer, restoreLoader string
var restoreRam int
var restoreNew bool

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [name]",
	Short: "Restore a backup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleRestoreBackup(args[0])
	},
}

func init() {
	backupRestoreCmd.Flags().StringVar(&restoreTarget, "target", "", "Target server ID (to restore to existing)")
	backupRestoreCmd.Flags().BoolVar(&restoreNew, "new", false, "Create new server from backup")
	backupRestoreCmd.Flags().StringVar(&restoreName, "name", "", "New server name")
	backupRestoreCmd.Flags().StringVar(&restoreVer, "version", "1.20.1", "New server version")
	backupRestoreCmd.Flags().StringVar(&restoreLoader, "loader", "vanilla", "New server loader")
	backupRestoreCmd.Flags().IntVar(&restoreRam, "ram", 2048, "New server RAM")

	backupCmd.AddCommand(backupCreateCmd, backupListCmd, backupDeleteCmd, backupRestoreCmd)
	RootCmd.AddCommand(backupCmd)
}

func handleBackupCreate(serverID, name string) error {
	resp, err := Client.CreateBackup(serverID, name)
	if err != nil {
		return fmt.Errorf("create backup for server %s: %w", serverID, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]string{
			"action":    "create_backup",
			"server_id": serverID,
			"status":    "ok",
			"message":   resp.Message,
			"path":      resp.Path,
		})
	}

	fmt.Printf("OK  backup created for server %s\n", serverID)
	fmt.Printf("Path: %s\n", resp.Path)
	return nil
}

func handleListBackups(serverID string) error {
	backups, err := Client.ListServerBackups(serverID)
	if err != nil {
		return fmt.Errorf("list backups for server %s: %w", serverID, err)
	}
	return printBackups(backups, serverID)
}

func handleListAllBackups() error {
	backups, err := Client.ListAllBackups()
	if err != nil {
		return fmt.Errorf("list backups: %w", err)
	}
	return printBackups(backups, "")
}

func printBackups(backups []sdk.BackupInfo, serverID string) error {
	if isJSONOutput() {
		type backupJSON struct {
			Name      string `json:"name"`
			SizeBytes int64  `json:"size_bytes"`
			SizeMB    string `json:"size_mb"`
		}

		items := make([]backupJSON, 0, len(backups))
		for _, b := range backups {
			items = append(items, backupJSON{
				Name:      b.Name,
				SizeBytes: b.Size,
				SizeMB:    formatMegabytes(b.Size),
			})
		}

		payload := struct {
			Action   string       `json:"action"`
			Status   string       `json:"status"`
			Backups  []backupJSON `json:"backups"`
			ServerID string       `json:"server_id,omitempty"`
		}{
			Action:   "list_backups",
			Status:   "ok",
			Backups:  items,
			ServerID: serverID,
		}

		return printJSON(payload)
	}

	rows := make([][]string, 0, len(backups))
	for _, b := range backups {
		rows = append(rows, []string{b.Name, formatMegabytes(b.Size), fmt.Sprintf("%d", b.Size)})
	}
	printTable([]string{"NAME", "SIZE_MB", "SIZE_BYTES"}, rows)
	return nil
}

func handleDeleteBackup(name string) error {
	if err := Client.DeleteBackup(name); err != nil {
		return fmt.Errorf("delete backup %s: %w", name, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]string{
			"action":      "delete_backup",
			"backup_name": name,
			"status":      "ok",
		})
	}

	fmt.Printf("OK  backup deleted: %s\n", name)
	return nil
}

func handleRestoreBackup(backupName string) error {
	req := sdk.RestoreBackupRequest{}

	if restoreNew {
		if restoreName == "" {
			return newValidationError("you must specify --name when using --new")
		}
		req.NewServerName = restoreName
		req.NewServerVersion = restoreVer
		req.NewServerLoader = restoreLoader
		req.NewServerRam = restoreRam
	} else {
		if restoreTarget == "" {
			return newValidationError("you must specify --target <ID> or use --new")
		}
		req.TargetServerID = restoreTarget
	}

	if err := Client.RestoreBackup(backupName, req); err != nil {
		return fmt.Errorf("restore backup %s: %w", backupName, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]string{
			"action":      "restore_backup",
			"backup_name": backupName,
			"status":      "ok",
		})
	}

	fmt.Printf("OK  backup restored: %s\n", backupName)
	return nil
}
