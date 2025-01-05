package internal

import (
	"context"
)

type EvalAgent struct {
	config *Config

	Role    string
	Message string
}

func NewEvalAgent(cfg *Config, role, content string) (*EvalAgent, error) {
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

	content, err := SendMessage(r.config, ctx, r.Role, message, input)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "EVAL",
		Content: content,
	}, nil
}
