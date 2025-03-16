package setup

import (
	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
)

var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up AI configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := internal.ParseConfig(args)
		if err != nil {
			internal.Exit(err)
		}
		agent.Setup(cfg)
	},
}

func init() {
	flags := SetupCmd.Flags()
	flags.String("editor", "vi", "Specify editor to use")

	flags.SortFlags = true
	SetupCmd.CompletionOptions.DisableDefaultCmd = true
}
