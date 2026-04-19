package cmd

import (
	"fmt"
	"log"

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
	Run: func(cmd *cobra.Command, args []string) {
		handleDeleteServer(args[0])
	},
}

var serverStartCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Start a server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleStartServer(args[0])
	},
}

var serverStopCmd = &cobra.Command{
	Use:   "stop [id]",
	Short: "Stop a server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		handleStopServer(args[0])
	},
}

func init() {
	serverCmd.AddCommand(serverDeleteCmd, serverStartCmd, serverStopCmd)
	RootCmd.AddCommand(serverCmd)
}

func handleDeleteServer(id string) {
	if err := Client.DeleteServer(id); err != nil {
		log.Fatalf("Error deleting server: %v", err)
	}
	fmt.Printf("Server %s deleted successfully.\n", id)
}

func handleStartServer(id string) {
	if err := Client.StartServer(id); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	fmt.Printf("Start command sent to server %s.\n", id)
}

func handleStopServer(id string) {
	if err := Client.StopServer(id); err != nil {
		log.Fatalf("Error stopping server: %v", err)
	}
	fmt.Printf("Stop command sent to server %s.\n", id)
}
