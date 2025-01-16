package agent

import (
	"context"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

// HelpAgent auto selects the best agent to handle the user query
type HelpAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewHelpAgent(cfg *llm.Config, role, content string) (*HelpAgent, error) {
	if role == "" {
		role = "system"
	}
	if content == "" {
		content = resource.GetAIHelpRoleContent()
	}

	cfg.Tools = llm.GetAIHelpTools()

	agent := HelpAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *HelpAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	var message = r.Message

	content, err := llm.Send(r.config, ctx, r.Role, message, input)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "AI",
		Content: content,
	}, nil
}
