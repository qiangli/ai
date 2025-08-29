package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qiangli/ai/cmd/agent"
	"github.com/qiangli/ai/cmd/history"
	// "github.com/qiangli/ai/cmd/hub"
	// "github.com/qiangli/ai/cmd/mcp"
	"github.com/qiangli/ai/cmd/setup"
	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
)

var viper = internal.V

const rootUsageTemplate = `AI Command Line Tool

Usage:
  ai [OPTIONS] [@AGENT] MESSAGE...{{if .HasExample}}

Examples:
{{.Example}}{{end}}

Miscellaneous:
  ai /setup                      Setup configuration

Use "{{.CommandPath}} /help [agents|commands|tools|info]" for more information.
`

const usageExample = `
ai what is fish
ai / what is fish
ai @ask what is fish
`

var agentCmd = agent.AgentCmd
var setupCmd = setup.SetupCmd

// var mcpCmd = mcp.McpCmd
var historyCmd = history.HistoryCmd

// var hubCmd = hub.HubCmd

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

// // init viper
// func initConfig() {
// 	defaultCfg := os.Getenv("AI_CONFIG")
// 	if defaultCfg == "" {
// 		if home, err := os.UserHomeDir(); err == nil {
// 			defaultCfg = filepath.Join(home, ".ai", "config.yaml")
// 		}
// 	}
// 	if defaultCfg != "" {
// 		viper.SetConfigFile(defaultCfg)
// 	}

// 	viper.AutomaticEnv()
// 	viper.SetEnvPrefix("ai")
// 	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
// 	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

// 	if err := viper.ReadInConfig(); err != nil {
// 		log.Debugf("Error reading config file: %s\n", err)
// 	}
// }

func main() {
	// cobra.OnInitialize(initConfig)
	internal.InitConfig(viper)

	args := os.Args

	// if no args and no input (piped), show help - short form
	// $ ai
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
	} else {
		// intercept builtin commands
		// $ ai /help [agents|commands|tools|info]
		//
		// $ ai /agent
		// $ ai /setup
		// $ ai /hub
		// $ ai /mcp
		// $ ai /history
		// $ ai /!<system-command>
		switch args[1] {
		case "/help":
			// agent detailed help
			// trigger built-in help command
			nArgs := append([]string{"--help"}, os.Args[1:]...)
			agentCmd.SetArgs(nArgs)
			// hack: for showing all config options as usual
			// cobra is not calling initConfig() for help command
			// initConfig()
			if err := agentCmd.Execute(); err != nil {
				internal.Exit(err)
			}
			return
		case "/agent":
			os.Args = os.Args[1:]
			if err := agentCmd.Execute(); err != nil {
				internal.Exit(err)
			}
			return
		case "/setup":
			os.Args = os.Args[1:]
			if err := setupCmd.Execute(); err != nil {
				internal.Exit(err)
			}
			return
		// case "/hub":
		// 	os.Args = os.Args[1:]
		// 	if err := hubCmd.Execute(); err != nil {
		// 		internal.Exit(err)
		// 	}
		// 	return
		// case "/mcp":
		// 	os.Args = os.Args[1:]
		// 	if err := mcpCmd.Execute(); err != nil {
		// 		internal.Exit(err)
		// 	}
		// 	return
		case "/history":
			os.Args = os.Args[1:]
			if err := historyCmd.Execute(); err != nil {
				internal.Exit(err)
			}
			return
		default:
			// /!<command> args...
			if strings.HasPrefix(args[1], "/!") {
				if len(args[1]) > 2 {
					out := runCommand(args[1][2:], os.Args[1:])
					log.Infoln(out)
				} else {
					log.Infoln("command not specified: /!<cmmand>")
				}
				return
			}
		}
	}

	// ai [/agent] ...
	// $ ai [@AGENT] MESSAGE...
	// $ ai [--agent AGENT] MESSAGE...
	if err := agentCmd.Execute(); err != nil {
		internal.Exit(err)
	}
}

func runCommand(bin string, args []string) string {
	cmd := exec.Command(bin, args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stderr.String()
	}
	return out.String()
}
