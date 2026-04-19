package cmd

import (
	"fmt"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute(port int) {
	RootCmd.PersistentFlags().StringVar(&BaseURL, "url", fmt.Sprintf("http://localhost:%d", port), "URL of the NaviServer Daemon")

	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
