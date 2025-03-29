package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

func RunAgent(cfg *internal.AppConfig) error {
	log.Debugf("Agent: %s %s %v\n", cfg.Agent, cfg.Command, cfg.Args)

	in, err := GetUserInput(cfg)
	if err != nil {
		return err
	}
	if in.IsEmpty() {
		return internal.NewUserInputError("no query provided")
	}

	in.Agent = cfg.Agent
	in.Command = cfg.Command
	return RunSwarm(cfg, in)
}

func HelpInfo(cfg *internal.AppConfig) error {
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
		log.Errorln(err)
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

func Setup(cfg *internal.AppConfig) error {
	if err := setupConfig(cfg); err != nil {
		log.Errorf("Error: %v\n", err)
		return err
	}
	return nil
}

func HelpAgents(cfg *internal.AppConfig) error {
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

func HelpCommands(cfg *internal.AppConfig) error {
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

func HelpTools(cfg *internal.AppConfig) error {

	const listTpl = `Available tools:

%s
Total: %v

Tools are used by agents to perform specific tasks. They are automatically selected based on the agent's capabilities and your input message.
`
	list := []string{}

	tools, err := listTools(cfg.McpServerUrl)
	if err != nil {
		return err
	}
	for _, v := range tools {
		list = append(list, fmt.Sprintf("%s: %s: %s\n", v.Type, v.ID(), strings.TrimSpace(v.Description)))
	}

	sort.Strings(list)

	log.Infof(listTpl, strings.Join(list, "\n"), len(list))
	return nil
}
