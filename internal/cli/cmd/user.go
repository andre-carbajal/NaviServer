package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"naviserver/pkg/sdk"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users and permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleUserList()
	},
}

var (
	userCreateUsername string
	userCreatePassword string
)

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a user",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		username := strings.TrimSpace(userCreateUsername)
		password := strings.TrimSpace(userCreatePassword)
		if username == "" {
			return newValidationError("you must specify --username")
		}
		if password == "" {
			return newValidationError("you must specify --password")
		}
		return handleUserCreate(username, password)
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete [user-id]",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleUserDelete(args[0])
	},
}

var userPasswordCmd = &cobra.Command{
	Use:   "password",
	Short: "Manage user password",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var userPasswordValue string

var userPasswordSetCmd = &cobra.Command{
	Use:   "set [user-id]",
	Short: "Set user password",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		password := strings.TrimSpace(userPasswordValue)
		if password == "" {
			return newValidationError("you must specify --password")
		}
		return handleUserPasswordSet(args[0], password)
	},
}

var userPermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Manage user permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var userPermissionsGetCmd = &cobra.Command{
	Use:   "get [user-id]",
	Short: "Get permissions for user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleUserPermissionsGet(args[0])
	},
}

var (
	userPermissionsServerID string
	userPermissionsPower    string
	userPermissionsConsole  string
)

var userPermissionsSetCmd = &cobra.Command{
	Use:   "set [user-id]",
	Short: "Set user permissions for a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID := strings.TrimSpace(userPermissionsServerID)
		if serverID == "" {
			return newValidationError("you must specify --server")
		}
		if !cmd.Flags().Changed("power") || !cmd.Flags().Changed("console") {
			return newValidationError("you must specify both --power and --console")
		}

		power, err := strconv.ParseBool(strings.TrimSpace(userPermissionsPower))
		if err != nil {
			return newValidationError("--power must be true or false")
		}

		console, err := strconv.ParseBool(strings.TrimSpace(userPermissionsConsole))
		if err != nil {
			return newValidationError("--console must be true or false")
		}

		if console && !power {
			power = true
		}

		return handleUserPermissionsSet(args[0], serverID, power, console)
	},
}

func init() {
	userCreateCmd.Flags().StringVar(&userCreateUsername, "username", "", "Username")
	userCreateCmd.Flags().StringVar(&userCreatePassword, "password", "", "Password")
	userPasswordSetCmd.Flags().StringVar(&userPasswordValue, "password", "", "New password")
	userPermissionsSetCmd.Flags().StringVar(&userPermissionsServerID, "server", "", "Server ID")
	userPermissionsSetCmd.Flags().StringVar(&userPermissionsPower, "power", "", "Can control server power (true|false)")
	userPermissionsSetCmd.Flags().StringVar(&userPermissionsConsole, "console", "", "Can view console and files (true|false)")

	userPasswordCmd.AddCommand(userPasswordSetCmd)
	userPermissionsCmd.AddCommand(userPermissionsGetCmd, userPermissionsSetCmd)

	userCmd.AddCommand(
		userListCmd,
		userCreateCmd,
		userDeleteCmd,
		userPasswordCmd,
		userPermissionsCmd,
	)

	RootCmd.AddCommand(userCmd)
}

func handleUserList() error {
	users, err := Client.ListUsers()
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "list_users",
			"status": "ok",
			"users":  users,
		})
	}

	rows := make([][]string, 0, len(users))
	for _, user := range users {
		rows = append(rows, []string{user.ID, user.Username, user.Role})
	}
	printTable([]string{"ID", "USERNAME", "ROLE"}, rows)
	return nil
}

func handleUserCreate(username, password string) error {
	user, err := Client.CreateUser(username, password)
	if err != nil {
		return fmt.Errorf("create user %s: %w", username, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action": "create_user",
			"status": "ok",
			"user":   user,
		})
	}

	fmt.Printf("OK  user created: %s\n", user.Username)
	fmt.Printf("ID: %s\n", user.ID)
	return nil
}

func handleUserDelete(userID string) error {
	if err := Client.DeleteUser(userID); err != nil {
		return fmt.Errorf("delete user %s: %w", userID, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":  "delete_user",
			"status":  "ok",
			"user_id": userID,
		})
	}

	fmt.Printf("OK  user deleted: %s\n", userID)
	return nil
}

func handleUserPasswordSet(userID, password string) error {
	if err := Client.UpdatePassword(userID, password); err != nil {
		return fmt.Errorf("set user password %s: %w", userID, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":  "set_user_password",
			"status":  "ok",
			"user_id": userID,
		})
	}

	fmt.Printf("OK  password updated for user: %s\n", userID)
	return nil
}

func handleUserPermissionsGet(userID string) error {
	permissions, err := Client.GetPermissions(userID)
	if err != nil {
		return fmt.Errorf("get user permissions %s: %w", userID, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":      "get_user_permissions",
			"status":      "ok",
			"user_id":     userID,
			"permissions": permissions,
		})
	}

	rows := make([][]string, 0, len(permissions))
	for _, permission := range permissions {
		rows = append(rows, []string{
			permission.ServerID,
			fmt.Sprintf("%t", permission.CanControlPower),
			fmt.Sprintf("%t", permission.CanViewConsole),
		})
	}

	printTable([]string{"SERVER_ID", "POWER", "CONSOLE"}, rows)
	return nil
}

func handleUserPermissionsSet(userID, serverID string, power, console bool) error {
	permissions, err := Client.GetPermissions(userID)
	if err != nil {
		return fmt.Errorf("get current permissions for user %s: %w", userID, err)
	}

	updated := false
	payload := make([]sdk.Permission, 0, len(permissions)+1)
	for _, permission := range permissions {
		if permission.ServerID == serverID {
			if !updated {
				payload = append(payload, sdk.Permission{
					UserID:          userID,
					ServerID:        serverID,
					CanControlPower: power,
					CanViewConsole:  console,
				})
				updated = true
			}
			continue
		}

		payload = append(payload, sdk.Permission{
			UserID:          userID,
			ServerID:        permission.ServerID,
			CanControlPower: permission.CanControlPower,
			CanViewConsole:  permission.CanViewConsole,
		})
	}

	if !updated {
		payload = append(payload, sdk.Permission{
			UserID:          userID,
			ServerID:        serverID,
			CanControlPower: power,
			CanViewConsole:  console,
		})
	}

	if err := Client.SetPermissions(payload); err != nil {
		return fmt.Errorf("set user permissions %s on server %s: %w", userID, serverID, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":  "set_user_permissions",
			"status":  "ok",
			"user_id": userID,
			"permission": map[string]any{
				"server_id": serverID,
				"power":     power,
				"console":   console,
			},
		})
	}

	fmt.Printf("OK  permissions updated for user %s on server %s\n", userID, serverID)
	return nil
}
