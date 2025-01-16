package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/qiangli/ai/internal/resource"
)

var aiHelpTools = []openai.ChatCompletionToolParam{
	define("ai_agent_info",
		"Get information about all supported AI agents",
		nil,
	),
	define("ai_agent_list",
		"List all available AI agents",
		nil,
	),
	define("ai_agent_help",
		"Get help for an AI agent",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"agent": map[string]string{
					"type":        "string",
					"description": "AI agent name",
				},
			},
			"required": []string{"agent"},
		}),
}

func runAIHelpTool(_ *ToolConfig, _ context.Context, name string, props map[string]interface{}) (string, error) {
	getStr := func(key string) (string, error) {
		return getStrProp(key, props)
	}

	switch name {
	case "ai_agent_info":
		return agentInfo()
	case "ai_agent_list":
		return listAgents()
	case "ai_agent_help":
		agent, err := getStr("agent")
		if err != nil {
			return "", err
		}
		if info, ok := resource.AgentInfo[agent]; ok {
			return info, nil
		}
		return "", fmt.Errorf("unknown AI agent: %s", agent)
	default:
		return "", fmt.Errorf("unknown AI tool: %s", name)
	}
}

func listAgents() (string, error) {
	agents := resource.AgentDesc
	var out []string
	for agent := range agents {
		out = append(out, agent)
	}
	return strings.Join(out, "\n"), nil
}

func agentInfo() (string, error) {
	info := resource.AgentInfo
	var out []string
	for agent, desc := range info {
		out = append(out, fmt.Sprintf("%s: %s", agent, desc))
	}
	return strings.Join(out, "\n"), nil
}

var helpAgentNames = []string{"ai_agent_info", "which", "command", "man", "uname"}

func GetAIHelpTools() []openai.ChatCompletionToolParam {
	return filteredTools(helpAgentNames)
}
