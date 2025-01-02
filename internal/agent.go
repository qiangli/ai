package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/openai/openai-go"
)

type Agent interface {
	Send(ctx context.Context, input string) (*ChatMessage, error)
}

type ChatMessage struct {
	Agent   string
	Content string
}

type Role = openai.ChatCompletionMessageParamRole

// https://platform.openai.com/docs/guides/text-generation#developer-messages
func buildRoleMessage(role string, content string) openai.ChatCompletionMessageParamUnion {
	switch role {
	case "system":
		return openai.SystemMessage(content)
	case "assistant":
		return openai.AssistantMessage(content)
	case "user":
		return openai.UserMessage(content)
	// case "tool":
	// 	return openai.ToolMessage("", content)
	// case "function":
	// 	return openai.FunctionMessage("", content)
	case "developer":
		// return DeveloperMessage(content)
		return openai.SystemMessage(content)
	default:
		return nil
	}
}

func DeveloperMessage(content string) openai.ChatCompletionMessageParamUnion {
	return openai.ChatCompletionDeveloperMessageParam{
		Role: openai.F(openai.ChatCompletionDeveloperMessageParamRoleDeveloper),
		Content: openai.F([]openai.ChatCompletionContentPartTextParam{
			openai.TextPart(content),
		}),
	}
}

var availableAgents = map[string]string{
	"ask":  "Ask a general question",
	"chat": "Simple chat",
	// "aider":      "AI pair programming in your terminal",
	// "openhands":  "A platform for software development agents powered by AI",
	// "vanna":      "Let Vanna.AI write your SQL for you",
	// "research": "Autonomous agent designed for comprehensive web and local research",
}

func ListAgents() (map[string]string, error) {
	return availableAgents, nil
}

func AvailableAgents() string {
	dict, _ := ListAgents()
	list := make([]string, 0, len(dict))
	for k, v := range dict {
		list = append(list, fmt.Sprintf("%s: %s", k, v))
	}
	sort.Strings(list)
	var result strings.Builder
	for _, v := range list {
		result.WriteString(fmt.Sprintf("%s\n", v))
	}
	return result.String()
}
