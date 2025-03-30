package main

import (
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/swarm/agent"
)

const rootUsageTemplate = `AI Command Line Tool

Usage:
  ai [OPTIONS] [@AGENT] MESSAGE...{{if .HasExample}}

Examples:
{{.Example}}{{end}}

Use "{{.CommandPath}} /help [agents|commands|tools|info]" for more information.
`

const usageExample = `
ai what is fish
ai / what is fish
ai @ask what is fish
`

const agentUsageTemplate = `AI Command Line Tool

Usage:
  ai [OPTIONS] [@AGENT] MESSAGE...{{if .HasExample}}
{{.Example}}{{end}}

Miscellaneous:

  ai MESSAGE...                  Auto select agent for help with any questions
  ai /[COMMAND]       MESSAGE... Use script agent for help with command and shell scripts
  ai @[AGENT/COMMAND] MESSAGE... Engage specialist agent for help with various tasks

  ai /mcp                        Manage MCP server
  ai /setup                      Setup configuration
{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Options:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .EnvNames}}
Environment variables:
  {{.EnvNames}}{{end}}

Use "{{.CommandPath}} /help [agents|commands|tools|info]" for more information.
`

const inputInstruction = `
There are multiple ways to interact with this AI tool.

+ Command line input:

  ai @agent what is fish?

+ Read from standard input:

  ai @agent --stdin
  ai @agent -

Ctrl+D to send, Ctrl+C to cancel.

+ Here document:

  ai @agent <<eof
what is the weather today?
eof

+ Piping input:

  git diff origin/main | ai @agent [message...]

+ File redirection:

  ai @agent [message...] < file.txt

+ Reading from system clipboard:

  ai @agent [message...] --pb-read
  ai @agent [message...] {
  ai @agent [message...] --pb-read-wait
  ai @agent [message...] {{

Use system copy (Ctrl+C on Unix) to add selected contents.
Ctrl+C to cancel.

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

func Help(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		cfg, err := internal.ParseConfig(args)
		if err != nil {
			return err
		}
		agent.InitApp(cfg)
		for _, v := range args {
			switch v {
			case "agents", "agent":
				return agent.HelpAgents(cfg)
			case "commands", "command":
				return agent.HelpCommands(cfg)
			case "tools", "tool":
				return agent.HelpTools(cfg)
			case "info":
				return agent.HelpInfo(cfg)
			}
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
	}).Parse(agentUsageTemplate)
	if err != nil {
		return err
	}

	if err := tpl.Execute(cmd.OutOrStdout(), data); err != nil {
		return nil
	}
	return nil
}
