package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/agent/conf"
	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

const agentUsageTemplate = `AI Command Line Tool

Usage:
  ai [OPTIONS] [@AGENT] MESSAGE...{{if .HasExample}}
{{.Example}}{{end}}
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
  ai @agent [message...] --pb-tail
  ai @agent [message...] {{

Use system copy (Ctrl+C on Unix) to add selected contents.
Ctrl+C to cancel.

+ Composing with text editor:

  export AI_EDITOR=nano # default: builtin
  ai @agent -e
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

	// AI_XXX env
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

func Help(ctx context.Context, cmd *cobra.Command, args []string) error {
	cfg, err := setupAppConfig(ctx, args)
	if err != nil {
		return err
	}
	log.GetLogger(ctx).SetLogLevel(api.ToLogLevel("info"))

	vars, err := agent.InitVars(cfg)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		for _, v := range args {
			switch v {
			case "agents", "agent":
				return HelpAgents(ctx, vars)
			case "commands", "command":
				return HelpCommands(ctx)
			case "tools", "tool":
				return HelpTools(ctx, vars)
			case "models", "model", "aliases", "alias":
				return HelpModels(ctx, vars)
			case "info":
				return HelpInfo(ctx, vars)
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

func HelpInfo(ctx context.Context, vars *api.Vars) error {
	const format = `System info:

%v

LLM:

Provider: %s
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

	ac, err := json.MarshalIndent(vars.Config, "", "  ")
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

	// cfg := vars.Config

	// TODO
	// log.GetLogger(ctx).Infof(format, info, cfg.LLM.Name, cfg.LLM.BaseUrl, cfg.LLM.ApiKey, string(ac), strings.Join(filteredEnvs, "\n"))
	// m, err := cfg.ModelLoader(api.Any)
	// if err != nil {
	// 	m = &api.Model{}
	// }
	m := &api.Model{}
	log.GetLogger(ctx).Infof(format, info, m.Provider, m.BaseUrl, "***", string(ac), strings.Join(filteredEnvs, "\n"))
	return nil
}

func HelpAgents(ctx context.Context, vars *api.Vars) error {
	const format = `Available agents:

%s
Total: %v

Usage:

ai [OPTIONS] @AGENT[/COMMAND] MESSAGE...
ai [OPTIONS] --agent AGENT[/COMMAND] MESSAGE...  Engage specialist agent for help with various tasks

or

ai [OPTIONS] MESSAGE...
ai [OPTIONS] MESSAGE... [@AGENT[/COMMAND]]

AI will choose an appropriate agent based on your message if no agent is specified.

* If you specify agents at both the beginning and end of a message, the last one takes precedence.
* You can place command options anywhere in your message. To include options as part of the message, use quotes or escape '\'.
`
	assets := conf.Assets(vars.Config)
	agents, _ := assets.ListAgent(vars.Config.User.Email)

	dict := make(map[string]*api.AgentConfig)
	for _, v := range agents {
		for _, sub := range v.Agents {
			dict[sub.Name] = sub
		}
	}

	keys := make([]string, 0)
	for k := range dict {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteString(":\t")
		buf.WriteString(dict[k].Description)
		buf.WriteString("\n\n")
	}

	log.GetLogger(ctx).Infof(format, buf.String(), len(keys))

	return nil
}

func HelpCommands(ctx context.Context) error {
	const listTpl = `Available commands on the system:

%s

Total: %v
`
	list := util.ListCommands()

	var commands []string
	for k, v := range list {
		commands = append(commands, fmt.Sprintf("%s: %s", k, v))
	}
	sort.Strings(commands)

	log.GetLogger(ctx).Infof(listTpl, strings.Join(commands, "\n"), len(commands))
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

func HelpTools(ctx context.Context, vars *api.Vars) error {

	const listTpl = `Available tools:

%s
Total: %v

Tools are used by agents to perform specific tasks. They are automatically selected based on the agent's capabilities and your input message.
`
	list := []string{}

	assets := conf.Assets(vars.Config)
	tools, _ := assets.ListToolkit(vars.Config.User.Email)

	for kit, tc := range tools {
		for _, v := range tc.Tools {
			list = append(list, fmt.Sprintf("%s:%s (%s) -- %s\n", kit, v.Name, v.Type, strings.TrimSpace(v.Description)))
		}
	}

	sort.Strings(list)

	log.GetLogger(ctx).Infof(listTpl, strings.Join(list, "\n"), len(list))
	return nil
}

func HelpModels(ctx context.Context, vars *api.Vars) error {

	const listTpl = `Available models:

%s
Total: %v

Model Alias can be used to reference a group of LLM models. You can mix and match different providers for one alias.
`
	list := []string{}

	assets := conf.Assets(vars.Config)
	models, _ := assets.ListModels(vars.Config.User.Email)

	for alias, tc := range models {
		var keys []string
		for k := range tc.Models {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var levels []string
		for _, k := range keys {
			v := tc.Models[k]
			levels = append(levels, fmt.Sprintf("    %s (%s)\n    %s\n    %s\n    %s\n", k, v.Provider, v.Model, v.BaseUrl, v.ApiKey))
		}
		list = append(list, fmt.Sprintf("%s:\n%s\n", alias, strings.Join(levels, "\n")))
	}

	sort.Strings(list)

	log.GetLogger(ctx).Infof(listTpl, strings.Join(list, "\n"), len(list))
	return nil
}
