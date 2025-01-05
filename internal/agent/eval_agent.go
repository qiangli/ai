package agent

import (
	"context"

	"github.com/qiangli/ai/internal/llm"
)

type EvalAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewEvalAgent(cfg *llm.Config, role, content string) (*EvalAgent, error) {
	if role == "" {
		role = "system"
	}

	agent := EvalAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *EvalAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	var message = r.Message

	content, err := llm.Send(r.config, ctx, r.Role, message, input)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "EVAL",
		Content: content,
	}, nil
}
