package agent

import (
	"context"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
)

type EvalAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string
}

func NewEvalAgent(cfg *internal.AppConfig) (*EvalAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt

	if role == "" {
		role = "system"
	}

	agent := EvalAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *EvalAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	content, err := llm.Send(r.config.LLM, ctx, r.Role, r.Prompt, in.Input())
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "EVAL",
		Content: content,
	}, nil
}
