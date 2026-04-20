package cmd

import (
	"errors"
	"fmt"
	"naviserver/pkg/sdk"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var serverDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleDeleteServer(args[0])
	},
}

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List servers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleListServers()
	},
}

var (
	createName    string
	createLoader  string
	createVersion string
	createRAM     int
	createAsync   bool
)

var serverCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(createName) == "" {
			return newValidationError("you must specify --name")
		}
		if createRAM <= 0 {
			return newValidationError("--ram must be greater than 0")
		}
		return handleCreateServer(createName, createLoader, createVersion, createRAM, createAsync)
	},
}

var serverStartCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Start a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleStartServer(args[0])
	},
}

var serverStopCmd = &cobra.Command{
	Use:   "stop [id]",
	Short: "Stop a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleStopServer(args[0])
	},
}

func init() {
	serverCreateCmd.Flags().StringVar(&createName, "name", "", "Server name")
	serverCreateCmd.Flags().StringVar(&createLoader, "loader", "vanilla", "Server loader")
	serverCreateCmd.Flags().StringVar(&createVersion, "version", "1.20.1", "Server version")
	serverCreateCmd.Flags().IntVar(&createRAM, "ram", 2048, "Server RAM in MB")
	serverCreateCmd.Flags().BoolVar(&createAsync, "async", false, "Run create request asynchronously")

	serverCmd.AddCommand(serverListCmd, serverCreateCmd, serverDeleteCmd, serverStartCmd, serverStopCmd)
	RootCmd.AddCommand(serverCmd)
}

func handleListServers() error {
	servers, err := Client.ListServers()
	if err != nil {
		return fmt.Errorf("list servers: %w", err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":  "list_servers",
			"status":  "ok",
			"servers": servers,
		})
	}

	rows := make([][]string, 0, len(servers))
	for _, s := range servers {
		rows = append(rows, []string{
			s.ID,
			s.Name,
			s.Status,
			fmt.Sprintf("%d", s.Port),
			s.Version,
			s.Loader,
			fmt.Sprintf("%d", s.RAM),
		})
	}

	printTable([]string{"ID", "NAME", "STATUS", "PORT", "VERSION", "LOADER", "RAM_MB"}, rows)
	return nil
}

func handleCreateServer(name, loader, version string, ram int, async bool) error {
	req := sdk.CreateServerRequest{
		Name:      name,
		Loader:    loader,
		Version:   version,
		Ram:       ram,
		RequestID: uuid.New().String(),
	}

	if async {
		if err := Client.CreateServer(req); err != nil {
			return fmt.Errorf("create server %s: %w", name, err)
		}

		if isJSONOutput() {
			return printJSON(map[string]any{
				"action":      "create_server",
				"status":      "accepted",
				"server_name": name,
				"request_id":  req.RequestID,
				"mode":        "async",
			})
		}

		fmt.Printf("OK  create command accepted: %s\n", name)
		fmt.Printf("Request ID: %s\n", req.RequestID)
		return nil
	}

	conn, err := connectProgressWS(req.RequestID)
	if err != nil {
		return fmt.Errorf("connect create progress stream: %w", err)
	}
	defer conn.Close()

	if err := Client.CreateServer(req); err != nil {
		return fmt.Errorf("create server %s: %w", name, err)
	}

	lastMsg, err := waitCreateCompletion(conn)
	if err != nil {
		return fmt.Errorf("create server %s: %w", name, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]any{
			"action":      "create_server",
			"status":      "ok",
			"server_name": name,
			"request_id":  req.RequestID,
			"mode":        "sync",
			"message":     lastMsg,
		})
	}

	fmt.Printf("OK  server created: %s\n", name)
	return nil
}

func connectProgressWS(requestID string) (*websocket.Conn, error) {
	wsURL, err := Client.GetWebSocketURL(fmt.Sprintf("/ws/progress/%s", requestID))
	if err != nil {
		return nil, err
	}

	header := http.Header{}
	header.Set("X-NaviServer-Client", "CLI")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		return conn, nil
	}

	time.Sleep(500 * time.Millisecond)
	conn, _, err = websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func waitCreateCompletion(conn *websocket.Conn) (string, error) {
	lastMessage := ""

	for {
		var event sdk.ProgressEvent
		if err := conn.ReadJSON(&event); err != nil {
			if lastMessage == "Server created successfully" {
				return lastMessage, nil
			}
			return "", fmt.Errorf("progress stream closed before completion: %w", err)
		}

		message := strings.TrimSpace(event.Message)
		if message != "" {
			lastMessage = message
		}

		if !isJSONOutput() {
			if event.Progress > 0 {
				fmt.Printf("[%.0f%%] %s\n", event.Progress, message)
			} else {
				fmt.Printf("[....] %s\n", message)
			}
		}

		if event.Progress < 0 || strings.HasPrefix(strings.ToLower(message), "error:") {
			if message == "" {
				message = "server creation failed"
			}
			return "", errors.New(message)
		}

		if message == "Server created successfully" {
			return message, nil
		}
	}
}

func handleDeleteServer(id string) error {
	if err := Client.DeleteServer(id); err != nil {
		return fmt.Errorf("delete server %s: %w", id, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]string{
			"action":    "delete_server",
			"server_id": id,
			"status":    "ok",
		})
	}

	fmt.Printf("OK  server deleted: %s\n", id)
	return nil
}

func handleStartServer(id string) error {
	if err := Client.StartServer(id); err != nil {
		return fmt.Errorf("start server %s: %w", id, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]string{
			"action":    "start_server",
			"server_id": id,
			"status":    "ok",
		})
	}

	fmt.Printf("OK  start command sent: %s\n", id)
	return nil
}

func handleStopServer(id string) error {
	if err := Client.StopServer(id); err != nil {
		return fmt.Errorf("stop server %s: %w", id, err)
	}

	if isJSONOutput() {
		return printJSON(map[string]string{
			"action":    "stop_server",
			"server_id": id,
			"status":    "ok",
		})
	}

	fmt.Printf("OK  stop command sent: %s\n", id)
	return nil
}
