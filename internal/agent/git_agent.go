package agent

import (
	"context"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

type GitAgent struct {
	config *llm.Config

	Role   string
	Prompt string
}

func NewGitAgent(cfg *llm.Config, role, prompt string) (*GitAgent, error) {
	if role == "" {
		role = "system"
	}
	if prompt == "" {
		prompt = resource.GetGitSystemRoleContent(cfg.Git.Short)
	}
	agent := GitAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *GitAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	content, err := llm.Send(r.config, ctx, r.Role, r.Prompt, in.Input())
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "GIT",
		Content: content,
	}, nil
}
