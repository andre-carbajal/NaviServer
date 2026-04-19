package cmd

import (
	"fmt"
	"log"
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
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		if len(args) > 1 {
			name = args[1]
		}
		handleBackupCreate(args[0], name)
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list [serverId]",
	Short: "List backups",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			handleListBackups(args[0])
		} else {
			handleListAllBackups()
		}
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a backup",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleDeleteBackup(args[0])
	},
}

var restoreTarget, restoreName, restoreVer, restoreLoader string
var restoreRam int
var restoreNew bool

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [name]",
	Short: "Restore a backup",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleRestoreBackup(args[0])
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

func handleBackupCreate(serverID, name string) {
	resp, err := Client.CreateBackup(serverID, name)
	if err != nil {
		log.Fatalf("Error creating backup: %v", err)
	}
	fmt.Println(resp.Message)
	fmt.Printf("Location: %s\n", resp.Path)
}

func handleListBackups(serverID string) {
	backups, err := Client.ListServerBackups(serverID)
	if err != nil {
		log.Fatalf("Error listing backups: %v", err)
	}
	printBackups(backups)
}

func handleListAllBackups() {
	backups, err := Client.ListAllBackups()
	if err != nil {
		log.Fatalf("Error listing backups: %v", err)
	}
	printBackups(backups)
}

func printBackups(backups []sdk.BackupInfo) {
	fmt.Println("Backups:")
	for _, b := range backups {
		fmt.Printf("- %s (%.2f MB)\n", b.Name, float64(b.Size)/1024/1024)
	}
}

func handleDeleteBackup(name string) {
	if err := Client.DeleteBackup(name); err != nil {
		log.Fatalf("Error deleting backup: %v", err)
	}
	fmt.Println("Backup deleted successfully.")
}

func handleRestoreBackup(backupName string) {
	req := sdk.RestoreBackupRequest{}

	if restoreNew {
		if restoreName == "" {
			log.Fatal("Error: You must specify --name for the new server")
		}
		req.NewServerName = restoreName
		req.NewServerVersion = restoreVer
		req.NewServerLoader = restoreLoader
		req.NewServerRam = restoreRam
	} else {
		if restoreTarget == "" {
			log.Fatal("Error: You must specify --target <ID> or use --new")
		}
		req.TargetServerID = restoreTarget
	}

	if err := Client.RestoreBackup(backupName, req); err != nil {
		log.Fatalf("Error restoring backup: %v", err)
	}
	fmt.Println("Backup restored successfully.")
}
