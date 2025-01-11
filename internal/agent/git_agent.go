package agent

import (
	"context"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

type GitAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewGitAgent(cfg *llm.Config, role, content string) (*GitAgent, error) {
	if role == "" {
		role = "system"
	}
	if content == "" {
		content = resource.GetGitSystemRoleContent(cfg.Git.Short)
	}
	agent := GitAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *GitAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	var message = r.Message

	content, err := llm.Send(r.config, ctx, r.Role, message, input)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "GIT",
		Content: content,
	}, nil
}
