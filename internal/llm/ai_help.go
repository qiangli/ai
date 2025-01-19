package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/resource"
)

var aiHelpTools = []openai.ChatCompletionToolParam{
	define("ai_agent_list",
		"List all supported AI agents",
		nil,
	),
	define("ai_agent_info",
		"Get information about an AI agent",
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

func runAIHelpTool(_ *internal.ToolConfig, _ context.Context, name string, props map[string]interface{}) (string, error) {
	getStr := func(key string) (string, error) {
		return getStrProp(key, props)
	}

	switch name {
	case "ai_agent_list":
		return agentList()
	case "ai_agent_info":
		agent, err := getStr("agent")
		if err != nil {
			return "", err
		}
		return agentInfo(agent)
	default:
		return "", fmt.Errorf("unknown AI tool: %s", name)
	}
}

func agentList() (string, error) {
	info := resource.AgentDesc
	var out []string
	for agent, desc := range info {
		out = append(out, fmt.Sprintf("%s: %s", agent, desc))
	}
	return strings.Join(out, "\n"), nil
}

func agentInfo(agent string) (string, error) {
	info := resource.AgentInfo
	commands := resource.AgentCommands

	if desc, ok := info[agent]; ok {
		if cmds, ok := commands[agent]; ok {
			return fmt.Sprintf("%s\nAvailable commands:\n%s", desc, cmds), nil
		}
		return fmt.Sprintf("%s\nAvailable commands: None", desc), nil
	}
	return "", fmt.Errorf("unknown AI agent: %s", agent)
}

func GetAIHelpTools() []openai.ChatCompletionToolParam {
	return append(aiHelpTools, GetRestrictedSystemTools()...)
}
