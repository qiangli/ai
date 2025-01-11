package agent

import (
	"context"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
)

type Agent interface {
	Send(ctx context.Context, input string) (*ChatMessage, error)
}

type ChatMessage struct {
	Agent   string
	Content string
}

func MakeAgent(name string, cfg *llm.Config, role, content string) (Agent, error) {
	switch name {
	case "ask":
		return NewAskAgent(cfg, role, content)
	case "eval":
		return NewEvalAgent(cfg, role, content)
	case "seek":
		return NewGptrAgent(cfg, role, content)
	case "sql":
		return NewSqlAgent(cfg, role, content)
	case "gptr":
		return NewGptrAgent(cfg, role, content)
	case "oh":
		return NewOhAgent(cfg, role, content)
	case "git":
		return NewGitAgent(cfg, role, content)
	case "code":
		return NewAiderAgent(cfg, role, content)
	default:
		return nil, internal.NewUserInputError("not supported yet: " + name)
	}
}

func agentList() (map[string]string, error) {
	return resource.AgentDesc, nil
}

func HandleCommand(cfg *llm.Config, role, content string) error {
	log.Debugf("Handle: %s %v\n", cfg.Command, cfg.Args)

	command := cfg.Command

	if command != "" {
		// $ ai /command
		if strings.HasPrefix(command, "/") {
			return SlashCommand(cfg, role, content)
		}

		// $ ai @agent
		if strings.HasPrefix(command, "@") {
			return AgentCommand(cfg, role, content)
		}
	}

	// auto select the best agent to handle the user query if there is message content
	// $ ai message...
	return AgentHelp(cfg, role, content)
}
