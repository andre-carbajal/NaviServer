package cmd

import (
	"fmt"
	"naviserver/internal/cli/ui"
	"naviserver/pkg/sdk"
	"os"

	"github.com/spf13/cobra"
)

var (
	Client  *sdk.Client
	BaseURL string
)

var RootCmd = &cobra.Command{
	Use:   "naviserver-cli",
	Short: "CLI for NaviServer Server Manager",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		Client = sdk.NewClient(BaseURL)
	},
	Run: func(cmd *cobra.Command, args []string) {
		for {
			serverID := ui.RunServerDashboard(Client)
			if serverID == "" {
				break
			}
			if !ui.RunLogs(Client, serverID) {
				break
			}
		}
	},
}

func Execute(port int) {
	RootCmd.PersistentFlags().StringVar(&BaseURL, "url", fmt.Sprintf("http://localhost:%d", port), "URL of the NaviServer Daemon")

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
