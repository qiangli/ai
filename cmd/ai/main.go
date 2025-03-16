package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/cmd/agent"
	"github.com/qiangli/ai/cmd/mcp"
	"github.com/qiangli/ai/cmd/setup"
	"github.com/qiangli/ai/internal"
	help "github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/log"
)

var RootCmd = &cobra.Command{
	Use:     "ai [OPTIONS] AGENT [message...]",
	Short:   "AI command line tool",
	Example: usageExample,
	Args:    cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return Help(cmd)
	},
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
		log.Printf("Error reading config file: %s\n", err)
	}
}

func init() {
	defaultCfg := os.Getenv("AI_CONFIG")
	// default: ~/.ai/config.yaml
	if defaultCfg == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			defaultCfg = filepath.Join(home, ".ai", "config.yaml")
		}
	}

	// add config to all root commands
	RootCmd.PersistentFlags().StringVar(&internal.ConfigFile, "config", defaultCfg, "config file")
	agent.AgentCmd.PersistentFlags().StringVar(&internal.ConfigFile, "config", defaultCfg, "config file")
	mcp.McpCmd.PersistentFlags().StringVar(&internal.ConfigFile, "config", defaultCfg, "config file")
	setup.SetupCmd.PersistentFlags().StringVar(&internal.ConfigFile, "config", defaultCfg, "config file")

	//
	RootCmd.Flags().SortFlags = true
	RootCmd.CompletionOptions.DisableDefaultCmd = true
	RootCmd.SetHelpTemplate(rootUsageTemplate)

	// usage template for sub commands except for agent
	mcp.McpCmd.SetUsageTemplate(commandUsageTemplate)
	setup.SetupCmd.SetUsageTemplate(commandUsageTemplate)
}

// all commands start with slash "/"; otherwise, it is considered as input message
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
			if err := RootCmd.Help(); err != nil {
				internal.Exit(err)
			}
			return
		}
	}

	// built in commands
	// $ ai /help [agents|commands|tools|info]
	// $ ai /version
	// $ ai /mcp
	// $ ai /setup
	if len(args) > 1 && args[1][0] == '/' {
		switch args[1] {
		case "/help":
			// help -- long form
			if err := showHelp(args); err != nil {
				internal.Exit(err)
			}
			return
		case "/version":
			log.Printf("AI version %s\n", internal.Version)
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
		}
	}

	// ai [AGENT] [message...]
	agent.AgentCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		return Help(cmd)
	})
	if err := agent.AgentCmd.Execute(); err != nil {
		internal.Exit(err)
	}
}

func showHelp(args []string) error {
	cfg, err := internal.ParseConfig(args)
	if err != nil {
		return err
	}

	// ai /help [agents|commands|tools|info]
	if len(args) > 2 {
		switch args[2] {
		case "agents", "agent":
			return help.HelpAgents(cfg)
		case "commands", "command":
			return help.HelpCommands(cfg)
		case "tools", "tool":
			return help.HelpTools(cfg)
		case "info":
			return help.HelpInfo(cfg)
		}
	}

	// ai /help with all supported agent flags
	return Help(agent.AgentCmd)
}
