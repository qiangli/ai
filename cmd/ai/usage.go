package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/agent/resource"
	"github.com/qiangli/ai/swarm/api"
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
		// agent.InitApp(cfg)
		for _, v := range args {
			switch v {
			case "agents", "agent":
				return HelpAgents(cfg)
			case "commands", "command":
				return HelpCommands(cfg)
			case "tools", "tool":
				return HelpTools(cfg)
			case "info":
				return HelpInfo(cfg)
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

func HelpInfo(cfg *api.AppConfig) error {
	const format = `System info:

%v

LLM:

Default Model: %s
Base URL: %s
API Key: %s

AI default configuration:

%v

AI Environment:

%v
`
	info, err := collectSystemInfo()
	if err != nil {
		return err
	}

	jc, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Get the current environment variables
	envs := os.Environ()
	var filteredEnvs []string
	for _, v := range envs {
		if strings.HasPrefix(v, "AI_") {
			filteredEnvs = append(filteredEnvs, v)
		}
	}
	sort.Strings(filteredEnvs)

	log.Infof(format, info, cfg.LLM.Model, cfg.LLM.BaseUrl, cfg.LLM.ApiKey, string(jc), strings.Join(filteredEnvs, "\n"))
	return nil
}

// func Setup(cfg *api.AppConfig) error {
// 	if err := setupConfig(cfg); err != nil {
// 		return err
// 	}
// 	return nil
// }

func HelpAgents(cfg *api.AppConfig) error {
	const format = `Available agents:

%s
Total: %v

Usage:

ai @agent message...

Not sure which agent to use? Simply enter your message and AI will choose the most appropriate one for you.
`
	dict := resource.AgentCommandMap
	var keys []string
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteString(":\t")
		buf.WriteString(dict[k].Description)
		buf.WriteString("\n")
	}
	log.Infof(format, buf.String(), len(keys))

	return nil
}

func HelpCommands(cfg *api.AppConfig) error {
	list := util.ListCommands()

	const listTpl = `Available commands on the system:

%s

Total: %v

Usage:

ai /command message...

/ is shorthand for  @script/
`
	commands := make([]string, len(list))
	for i, v := range list {
		commands[i] = fmt.Sprintf("%s: %s", v[0], strings.TrimSpace(v[1]))
	}
	sort.Strings(commands)
	log.Infof(listTpl, strings.Join(commands, "\n"), len(commands))
	return nil
}

func collectSystemInfo() (string, error) {
	info, err := util.CollectSystemInfo()
	if err != nil {
		return "", err
	}
	jd, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jd), nil
}

func HelpTools(cfg *api.AppConfig) error {

	const listTpl = `Available tools:

%s
Total: %v

Tools are used by agents to perform specific tasks. They are automatically selected based on the agent's capabilities and your input message.
`
	list := []string{}

	vars, err := swarm.InitVars(cfg)
	if err != nil {
		return err
	}
	tools := vars.ListTools()

	for _, v := range tools {
		list = append(list, fmt.Sprintf("%s: %s: %s\n", v.Type, v.ID(), strings.TrimSpace(v.Description)))
	}

	sort.Strings(list)

	log.Infof(listTpl, strings.Join(list, "\n"), len(list))
	return nil
}
