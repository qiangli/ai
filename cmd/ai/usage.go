package main

import (
	"sort"
	"strings"
	"text/template"

	"github.com/qiangli/ai/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/qiangli/ai/internal/agent"
)

const rootUsageTemplate = `Usage:
  ai [OPTIONS] [@agent] message...{{if .HasExample}}

Examples:
{{.Example}}{{end}}

{{.CommandPath}} /help [info|agents|commands|tools] for more details.
`

const helpTemplate = `AI Command Line Tool

Usage:
  ai [OPTIONS] @agent message...{{if .HasExample}}
{{.Example}}{{end}}

Miscellaneous:

  ai message...                  Auto select agent for help with any questions
  ai /[command]       message... Use script agent for help with command and shell scripts
  ai @[agent/command] message... Engage specialist agent for help with various tasks

  ai /setup                      Setup AI configuration{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Options:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .EnvNames}}
Environment variables:
  {{.EnvNames}}{{end}}
`

const usageExample = `
ai what is fish
ai / what is fish
ai @ask what is fish
`

const inputInstruction = `
There are multiple ways to interact with this AI tool.

+ Command line input:

  ai @agent what is fish?

+ Read from standard input:

  ai @agent -
  ai @agent --stdin
Ctrl+D to send, Ctrl+C to cancel.

+ Here document:

  ai @agent <<eof
what is the weather today?
eof

+ Piping input:

  echo "What is Unix?" | ai @agent [message...]
  git diff origin/main | ai @agent [message...]

+ File redirection:

  ai @agent [message...] < file.txt

+ Reading from system clipboard:

  ai @agent [message...] {
Use system copy (Ctrl+C on Unix) to send selected contents.
Ctrl+C to cancel.

  ai @agent [message...] {{
Use system copy (Ctrl+C on Unix) to copy and press enter or "Y" to send.
Ctrl+C or enter "N" to cancel

+ Composing with text editor:

  export AI_EDITOR=nano # default: vi
  ai @agent
`

type HelpData struct {
	CommandPath string

	HasExample                 bool
	Example                    string
	HasAvailableLocalFlags     bool
	HasAvailableInheritedFlags bool
	LocalFlags                 *pflag.FlagSet
	InheritedFlags             *pflag.FlagSet
	EnvNames                   string
}

func getHelpData(cmd *cobra.Command) *HelpData {
	const envPrefix = "AI_"

	flagToEnv := make(map[string]string)
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		n := envPrefix + strings.ToUpper(strings.ReplaceAll(flag.Name, "-", "_"))
		flagToEnv[flag.Name] = n
	})

	names := make([]string, 0, len(flagToEnv))
	for _, v := range flagToEnv {
		names = append(names, v)
	}
	sort.Strings(names)

	localFlags := cmd.LocalFlags()
	inheritedFlags := cmd.InheritedFlags()

	return &HelpData{
		CommandPath:                cmd.CommandPath(),
		HasExample:                 len(inputInstruction) > 0,
		Example:                    inputInstruction,
		HasAvailableLocalFlags:     localFlags.HasAvailableFlags(),
		HasAvailableInheritedFlags: inheritedFlags.HasAvailableFlags(),
		LocalFlags:                 localFlags,
		InheritedFlags:             inheritedFlags,
		EnvNames:                   strings.Join(names, ", "),
	}
}

func Help(cfg *internal.AppConfig, cmd *cobra.Command) error {
	if len(cfg.Args) > 0 {
		switch cfg.Args[0] {
		case "agents", "agent":
			return agent.ListAgents(cfg)
		case "commands", "command":
			return agent.ListCommands(cfg)
		case "tools", "tool":
			return agent.ListTools(cfg)
		case "info":
			return agent.Info(cfg)
		}
	}

	// help
	trimTrailingWhitespaces := func(s string) string {
		return strings.TrimRightFunc(s, func(r rune) bool {
			return r == ' ' || r == '\t'
		})
	}

	data := getHelpData(cmd)

	tpl, err := template.New("help").Funcs(template.FuncMap{
		"trimTrailingWhitespaces": trimTrailingWhitespaces,
	}).Parse(helpTemplate)
	if err != nil {
		return err
	}

	if err := tpl.Execute(cmd.OutOrStdout(), data); err != nil {
		return nil
	}
	return nil
}
