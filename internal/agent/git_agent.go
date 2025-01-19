package agent

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

type GitAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string
}

func NewGitAgent(cfg *internal.AppConfig) (*GitAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt

	if role == "" {
		role = "system"
	}
	agent := GitAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *GitAgent) getSystemPrompt(in *UserInput) (string, error) {
	if r.Prompt != "" {
		return r.Prompt, nil
	}
	switch in.Subcommand {
	case "":
		return resource.GetGitSystemRoleContent(false), nil
	case "short":
		return resource.GetGitSystemRoleContent(true), nil
	case "conventional":
		return resource.GetGitSystemRoleContent(false), nil
	}
	return "", fmt.Errorf("unknown @git subcommand: %s", in.Subcommand)
}

func (r *GitAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	prompt, err := r.getSystemPrompt(in)
	if err != nil {
		return nil, err
	}

	content, err := llm.Send(r.config.LLM, ctx, r.Role, prompt, in.Input())
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "GIT",
		Content: content,
	}, nil
}
