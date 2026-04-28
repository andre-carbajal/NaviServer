package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long: "Generate shell completion scripts for bash, zsh, fish, and powershell.\n\n" +
		"Examples:\n" +
		"  naviserver-cli completion bash > ~/.local/share/bash-completion/completions/naviserver-cli\n" +
		"  naviserver-cli completion zsh > ~/.zfunc/_naviserver-cli\n" +
		"  naviserver-cli completion fish > ~/.config/fish/completions/naviserver-cli.fish\n" +
		"  naviserver-cli completion powershell > naviserver-cli.ps1",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenBashCompletionV2(os.Stdout, true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenZshCompletion(os.Stdout)
	},
}

var completionFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenFishCompletion(os.Stdout, true)
	},
}

var completionPowerShellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate PowerShell completion",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(
		completionBashCmd,
		completionZshCmd,
		completionFishCmd,
		completionPowerShellCmd,
	)

	RootCmd.AddCommand(completionCmd)
}
