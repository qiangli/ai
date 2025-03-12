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
	"github.com/qiangli/ai/internal/swarm"
	"github.com/qiangli/ai/internal/util"
)

func AgentHelp(cfg *internal.AppConfig) error {
	log.Debugf("Agent: %s\n", cfg.Agent)

	in, err := GetUserInput(cfg)
	if err != nil {
		return err
	}
	if in.IsEmpty() {
		return internal.NewUserInputError("no query provided")
	}

	in.Agent = cfg.Agent

	return RunSwarm(cfg, in)
}

func Info(cfg *internal.AppConfig) error {
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

	log.Printf(format, info, cfg.LLM.Model, cfg.LLM.BaseUrl, cfg.LLM.ApiKey, string(jc), strings.Join(filteredEnvs, "\n"))
	return nil
}

func Setup(cfg *internal.AppConfig) error {
	if err := setupConfig(cfg); err != nil {
		log.Errorf("Error: %v\n", err)
		return err
	}
	return nil
}

func ListAgents(cfg *internal.AppConfig) error {
	const format = `Available agents:

%s
Total: %v

Usage:

ai @agent message...

Not sure which agent to use? Simply enter your message and AI will choose the most appropriate one for you:

ai message...
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
	log.Printf(format, buf.String(), len(keys))

	return nil
}

func ListCommands(cfg *internal.AppConfig) error {
	list, err := util.ListCommands(false)
	if err != nil {
		log.Errorf("Error: %v\n", err)
		return err
	}

	const listTpl = `Available commands on the system:

%s

Total: %v

Usage:

ai /command message...

/ is shorthand for  @script/
`
	sort.Strings(list)
	log.Printf(listTpl, strings.Join(list, "\n"), len(list))
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

func ListTools(cfg *internal.AppConfig) error {

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
		list = append(list, fmt.Sprintf("%s: %s: %s\n", v.Label, v.Name(), strings.TrimSpace(v.Description)))
	}

	sort.Strings(list)

	log.Printf(listTpl, strings.Join(list, "\n"), len(list))
	return nil
}

func listTools(mcpServerUrl string) ([]*swarm.ToolFunc, error) {
	list := []*swarm.ToolFunc{}

	agentTools, err := ListAgentTools()
	if err != nil {
		return nil, err
	}
	list = append(list, agentTools...)

	// mcp tools
	mcpTools, err := swarm.ListMcpTools(mcpServerUrl)
	if err != nil {
		return nil, err
	}
	for _, v := range mcpTools {
		list = append(list, v...)
	}

	// system tools
	sysTools, err := swarm.ListSystemTools()
	if err != nil {
		return nil, err
	}
	list = append(list, sysTools...)

	return list, nil
}

func HandleCommand(cfg *internal.AppConfig) error {
	log.Debugf("HandleCommand: %s %v\n", cfg.Command, cfg.Args)

	cmd := cfg.Command

	if cmd != "" {
		// $ ai /command
		if strings.HasPrefix(cmd, "/") {
			name := strings.TrimSpace(cmd[1:])
			in, err := GetUserInput(cfg)
			if err != nil {
				return err
			}

			if name == "" && in.IsEmpty() {
				return internal.NewUserInputError("no command and message provided")
			}

			in.Agent = "script"
			in.Subcommand = name
			return RunSwarm(cfg, in)
		}

		// $ ai @agent
		if strings.HasPrefix(cmd, "@") {
			name := strings.TrimSpace(cmd[1:])
			if name == "" {
				// auto select
				// $ ai @ message...
				return AgentHelp(cfg)
			}

			in, err := GetUserInput(cfg)
			if err != nil {
				return err
			}
			if in.IsEmpty() {
				return internal.NewUserInputError("no message content")
			}

			in.Agent = name
			return RunSwarm(cfg, in)
		}
	}

	// auto select the best agent to handle the user query if there is message content
	// $ ai message...
	return AgentHelp(cfg)
}
