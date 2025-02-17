package main

import (
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const rootUsageTemplate = `Usage:
  ai message...
  ai [OPTIONS] AGENT [message...]{{if .HasExample}}

Examples:
{{.Example}}{{end}}

Agent:
  /[command]       [message...] Get help with system command and shell scripting tasks
  @[agent/command] [message...] Engage specialist agents for assistance with complex tasks

Use "{{.CommandPath}} help" for more info.
`

const helpTemplate = `AI Command Line Tool

Usage:
  ai message...
  ai [OPTIONS] AGENT [message...]{{if .Hint}}

{{.Hint}}{{end}}{{if .HasExample}}
{{.Example}}{{end}}

Agent:
  /[command]       [message...] Get help with system command and shell scripting tasks
  @[agent/command] [message...] Engage specialist agents for assistance with complex tasks

Miscellaneous:
  /                       List system commands available in the path
  @                       List all supported agents
  setup                   Setup the AI configuration{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Options:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .EnvNames}}
Environment variables:
  {{.EnvNames}}{{end}}
`

const usageExample = `
ai what is fish?
ai / what is fish?
ai @ask what is fish?
`

const usageHint = `
. Ask for help with writing or debugging shell scripts.
. Request explanations for specific shell commands or scripts.
. Get assistance with writing, optimizing, or debugging SQL queries.
. Seek guidance on writing code or debugging in various programming languages.
`

const inputInstruction = `
There are multiple ways to interact with the AI tool.

+ Command line input:

  ai AGENT what is fish?

+ Read from standard input:

  ai AGENT -
Ctrl+D to send, Ctrl+C to cancel.

+ Here document:

  ai AGENT <<eof
what is the weather today?
eof

+ Piping input:

  echo "What is Unix?" | ai AGENT
  git diff origin main | ai AGENT [message...]
  curl -sL https://en.wikipedia.org/wiki/Artificial_intelligence | head -100 | ai AGENT [message...]

+ File redirection:

  ai AGENT [message...] < file.txt

+ Reading from system clipboard:

  ai AGENT [message...] =
Use system copy (Ctrl+C on Unix) to send selected contents.
Ctrl+C to cancel.

+ Composing with text editor:

  export AI_EDITOR=nano # default: vi
  ai AGENT
`

type HelpData struct {
	CommandPath string

	Hint                       string
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
		Hint:                       usageHint,
		HasExample:                 len(inputInstruction) > 0,
		Example:                    inputInstruction,
		HasAvailableLocalFlags:     localFlags.HasAvailableFlags(),
		HasAvailableInheritedFlags: inheritedFlags.HasAvailableFlags(),
		LocalFlags:                 localFlags,
		InheritedFlags:             inheritedFlags,
		EnvNames:                   strings.Join(names, ", "),
	}
}

func Help(cmd *cobra.Command) error {
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
