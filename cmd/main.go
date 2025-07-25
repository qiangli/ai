package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/cmd/agent"
	"github.com/qiangli/ai/cmd/history"
	"github.com/qiangli/ai/cmd/mcp"
	"github.com/qiangli/ai/cmd/setup"
	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
)

const rootUsageTemplate = `AI Command Line Tool

Usage:
  ai [OPTIONS] [@AGENT] MESSAGE...{{if .HasExample}}

Examples:
{{.Example}}{{end}}

Miscellaneous:
  ai /mcp                        Manage MCP server
  ai /setup                      Setup configuration

Use "{{.CommandPath}} /help [agents|commands|tools|info]" for more information.
`

const usageExample = `
ai what is fish
ai / what is fish
ai @ask what is fish
`

var rootCmd = &cobra.Command{
	Use:                   "ai [OPTIONS] [@AGENT] MESSAGE...",
	Short:                 "AI Command Line Tool",
	Example:               usageExample,
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
	Args:                  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.SetHelpTemplate(rootUsageTemplate)
}

func initConfig() {
	if internal.ConfigFile != "" {
		viper.SetConfigFile(internal.ConfigFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			internal.Exit(err)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".ai")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Debugf("Error reading config file: %s\n", err)
	}
}

func main() {
	cobra.OnInitialize(initConfig)

	args := os.Args

	// $ ai
	// if no args and no input, show help - short form
	if len(args) <= 1 {
		isPiped := func() bool {
			stat, _ := os.Stdin.Stat()
			return (stat.Mode() & os.ModeCharDevice) == 0
		}
		if !isPiped() {
			if err := rootCmd.Execute(); err != nil {
				internal.Exit(err)
			}
			return
		}
	}

	// intercept custom commands
	// $ ai /help [agents|commands|tools|info]
	// $ ai /mcp
	// $ ai /setup
	// $ ai /history
	switch args[1] {
	case "/help":
		// agent detailed help
		// trigger built-in help command
		nArgs := append([]string{"--help"}, os.Args[1:]...)
		agent.AgentCmd.SetArgs(nArgs)
		// hack: for showing all config options as usual
		// cobra is not calling initConfig() for help command
		initConfig()
		if err := agent.AgentCmd.Execute(); err != nil {
			internal.Exit(err)
		}
		return
	case "/mcp":
		os.Args = os.Args[1:]
		if err := mcp.McpCmd.Execute(); err != nil {
			internal.Exit(err)
		}
		return
	case "/setup":
		os.Args = os.Args[1:]
		if err := setup.SetupCmd.Execute(); err != nil {
			internal.Exit(err)
		}
		return
	case "/history":
		os.Args = os.Args[1:]
		if err := history.HistoryCmd.Execute(); err != nil {
			internal.Exit(err)
		}
		return
	}

	// ai /ai ...
	// $ ai [@AGENT] MESSAGE...
	if err := agent.AgentCmd.Execute(); err != nil {
		internal.Exit(err)
	}
}
