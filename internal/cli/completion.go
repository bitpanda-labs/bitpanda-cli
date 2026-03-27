package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func (app *App) registerCompletion(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for bp.

To load completions:

  # Bash
  source <(bp completion bash)

  # Zsh
  source <(bp completion zsh)

  # Fish
  bp completion fish | source

  # PowerShell
  bp completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return parent.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return parent.GenZshCompletion(os.Stdout)
			case "fish":
				return parent.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return parent.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}

	parent.AddCommand(cmd)
}
