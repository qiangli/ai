package main

import (
	"context"
	"os"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
)

// const rootUsageTemplate = `AI Command Line Tool

// Usage:
//   ai [OPTIONS] [@AGENT] MESSAGE...{{if .HasExample}}

// Examples:
// {{.Example}}{{end}}

// Use "{{.CommandPath}} /help [agents|tools|models]" for more information.
// `

// const usageExample = `
// ai what is fish
// ai @ask what is fish
// `

// var rootCmd = &cobra.Command{
// 	Use:                   "ai [OPTIONS] [@AGENT] MESSAGE...",
// 	Short:                 "AI Command Line Tool",
// 	Example:               usageExample,
// 	DisableFlagsInUseLine: true,
// 	DisableSuggestions:    true,
// 	Args:                  cobra.ArbitraryArgs,
// 	RunE: func(cmd *cobra.Command, args []string) error {
// 		return cmd.Help()
// 	},
// }

// func init() {
// 	rootCmd.SetHelpTemplate(rootUsageTemplate)
// }

func main() {
	ctx := context.TODO()

	args := os.Args

	// support execution of ai script file (.sh or .yaml)
	shebang := strings.HasSuffix(args[0], ".yaml") || strings.HasSuffix(args[0], ".sh")

	// // if no args and no input (piped), show help - short form
	// // $ ai
	// if len(args) <= 1 && !shebang {
	// 	if err := rootCmd.Execute(); err != nil {
	// 		internal.Exit(ctx, err)
	// 	}
	// 	return
	// }

	// shebang support
	if shebang {
		args = append([]string{"/sh:bash", "--script", args[0]}, args[1:]...)
	} else {
		if len(args) <= 1 {
			args = []string{"/help:help"}
		}
	}

	// slash commands
	// intercept builtin commands
	// $ ai /help [agents|commands|tools|info]
	//
	// if strings.HasPrefix(args[1], "/") {
	// 	switch args[1] {
	// 	case "/help":
	// 		err := agent.Help(ctx, args)
	// 		if err != nil {
	// 			internal.Exit(ctx, err)
	// 		}
	// 		return
	// 	default:

	// 	}
	// }

	if err := agent.Run(ctx, args); err != nil {
		internal.Exit(ctx, err)
	}
}
