package agent

import (
	"context"

	"github.com/qiangli/ai/internal/llm"
)

type EvalAgent struct {
	config *llm.Config

	Role   string
	Prompt string
}

func NewEvalAgent(cfg *llm.Config, role, prompt string) (*EvalAgent, error) {
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
	content, err := llm.Send(r.config, ctx, r.Role, r.Prompt, in.Input())
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "EVAL",
		Content: content,
	}, nil
}
