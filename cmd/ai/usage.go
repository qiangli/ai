package main

import (
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/qiangli/ai/internal/resource"
)

const rootUsageTemplate = `Usage:
  ai message...
  ai [OPTIONS] COMMAND [message...]{{if .HasExample}}

Examples:

{{.Example}}{{end}}

Commands:
  /[binary] [message...]  Get help with executable and shell scripting tasks
  @[agent]  [message...]  Engage agents for help with various tasks

Use "{{.CommandPath}} help" for more info.
`

const helpTemplate = `AI Command Line Tool

Usage:
  ai message...
  ai [OPTIONS] COMMAND [message...]{{if .Hint}}

{{.Hint}}{{end}}{{if .HasExample}}

{{.Example}}{{end}}

Commands:
  /[binary] [message...]  Get help with executable and shell scripting tasks
  @[agent]  [message...]  Engage agents for help with various tasks

Miscellaneous:
  info                    Show system information shared with LLM
  setup                   Setup the AI configuration

Supported Agent:
  ask                     Ask general questions
  sql                     Generate SQL queries for your dataset
  eval                    Send direct messages without system prompts for evaluation{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Options:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .EnvNames}}
Environment variables:
  {{.EnvNames}}{{end}}
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

	hint := resource.GetUserHint()
	ex := resource.GetUserInputInstruction()

	localFlags := cmd.LocalFlags()
	inheritedFlags := cmd.InheritedFlags()

	return &HelpData{
		CommandPath:                cmd.CommandPath(),
		Hint:                       hint,
		HasExample:                 len(ex) > 0,
		Example:                    ex,
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
