package main

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:
  $ source <(cm completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ cm completion bash > /etc/bash_completion.d/cm
  # macOS:
  $ cm completion bash > /usr/local/etc/bash_completion.d/cm

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ cm completion zsh > "${fpath[1]}/_cm"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ cm completion fish | source

  # To load completions for each session, execute once:
  $ cm completion fish > ~/.config/fish/completions/cm.fish

PowerShell:
  PS> cm completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, execute once:
  PS> cm completion powershell > cm.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
