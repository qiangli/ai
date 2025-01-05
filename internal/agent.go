package internal

import (
	"context"

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
	"eval": "Direct message to AI",
	"seek": "Explore the web for information",
	"gptr": "GPT Researcher",

	// "aider":      "AI pair programming in your terminal",
	// "openhands":  "A platform for software development agents powered by AI",
	// "vanna":      "Let Vanna.AI write your SQL for you",
}

func MakeAgent(name string, cfg *Config, role, content string) (Agent, error) {
	switch name {
	case "ask":
		return NewAskAgent(cfg, role, content)
	case "eval":
		return NewEvalAgent(cfg, role, content)
	case "seek":
		return NewSeekAgent(cfg, role, content)
	case "gptr":
		return NewGptrAgent(cfg, role, content)
	default:
		return nil, NewUserInputError("not supported yet: " + name)
	}
}

func ListAgents() (map[string]string, error) {
	return availableAgents, nil
}
