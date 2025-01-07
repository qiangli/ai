package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/qiangli/ai/internal/resource"
)

var AIHelpTools = []openai.ChatCompletionToolParam{
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

func runAIHelpTool(_ *Config, _ context.Context, name string, props map[string]interface{}) (string, error) {
	getStr := func(key string) (string, error) {
		val, ok := props[key]
		if !ok {
			return "", fmt.Errorf("missing property: %s", key)
		}
		str, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("property %s must be a string", key)
		}
		return str, nil
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
	agents := resource.AgentList
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

var AllTools = append(append(AIHelpTools, SystemTools...), DBTools...)

var helpAgentNames = []string{"ai_agent_info", "which", "command", "man", "help", "uname"}

func filteredTools(names []string) []openai.ChatCompletionToolParam {
	// filter AllTools by function name
	var tools []openai.ChatCompletionToolParam
	for _, tool := range AllTools {
		for _, name := range names {
			if tool.Function.Value.Name.Value == name {
				tools = append(tools, tool)
			}
		}
	}
	return tools
}

func HelpAgentTools() []openai.ChatCompletionToolParam {
	return filteredTools(helpAgentNames)
}
