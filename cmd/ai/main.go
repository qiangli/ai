package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/cmd/mcp"
	"github.com/qiangli/ai/cmd/setup"
	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
)

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
		log.Errorf("Error reading config file: %s\n", err)
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
	for _, arg := range args {
		switch arg {
		case "/help":
			// trigger built-in help command
			args = append([]string{"--help"}, os.Args[1:]...)
			AgentCmd.SetArgs(args)
			if err := AgentCmd.Execute(); err != nil {
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
		}
	}

	// $ ai [@AGENT] MESSAGE...
	if err := AgentCmd.Execute(); err != nil {
		internal.Exit(err)
	}
}
