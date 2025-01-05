package agent

import (
	"context"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
)

type Agent interface {
	Send(ctx context.Context, input string) (*ChatMessage, error)
}

type ChatMessage struct {
	Agent   string
	Content string
}

// type Role = openai.ChatCompletionMessageParamRole

// func DeveloperMessage(content string) openai.ChatCompletionMessageParamUnion {
// 	return openai.ChatCompletionDeveloperMessageParam{
// 		Role: openai.F(openai.ChatCompletionDeveloperMessageParamRoleDeveloper),
// 		Content: openai.F([]openai.ChatCompletionContentPartTextParam{
// 			openai.TextPart(content),
// 		}),
// 	}
// }

var availableAgents = map[string]string{
	"ask":  "Ask a general question",
	"eval": "Direct message to AI",
	"seek": "Explore the web for information",
	"gptr": "GPT Researcher",
	"sql":  "SQL Assistant",

	// "aider":      "AI pair programming in your terminal",
	// "openhands":  "A platform for software development agents powered by AI",
	// "vanna":      "Let Vanna.AI write your SQL for you",
}

func MakeAgent(name string, cfg *llm.Config, role, content string) (Agent, error) {
	switch name {
	case "ask":
		return NewAskAgent(cfg, role, content)
	case "eval":
		return NewEvalAgent(cfg, role, content)
	case "seek":
		return NewSeekAgent(cfg, role, content)
	case "gptr":
		return NewGptrAgent(cfg, role, content)
	case "sql":
		return NewSqlAgent(cfg, role, content)
	default:
		return nil, internal.NewUserInputError("not supported yet: " + name)
	}
}

func ListAgents() (map[string]string, error) {
	return availableAgents, nil
}
