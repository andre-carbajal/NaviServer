package cmd

import (
	"fmt"
	"naviserver/pkg/sdk"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	Client       *sdk.Client
	BaseURL      string
	outputFormat string
)

var RootCmd = &cobra.Command{
	Use:           "naviserver-cli",
	Short:         "CLI for NaviServer Server Manager",
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		outputFormat = strings.ToLower(strings.TrimSpace(outputFormat))

		switch outputFormat {
		case "table", "json":
		default:
			return newValidationError("invalid --output value; use 'table' or 'json'")
		}

		Client = sdk.NewClient(BaseURL)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute(port int) {
	RootCmd.PersistentFlags().StringVar(&BaseURL, "url", fmt.Sprintf("http://localhost:%d", port), "URL of the NaviServer Daemon")
	RootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format: table|json")

	if err := RootCmd.Execute(); err != nil {
		printCommandError(err)
		os.Exit(commandExitCode(err))
	}
}
