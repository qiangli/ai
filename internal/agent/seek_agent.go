package agent

import (
	"context"

	"github.com/qiangli/ai/internal/llm"
)

type SeekAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewSeekAgent(cfg *llm.Config, role, content string) (*SeekAgent, error) {
	if role == "" {
		role = "system"
	}

	agent := SeekAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *SeekAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	// echo back the input for now
	content := input

	return &ChatMessage{
		Agent:   "SEEK",
		Content: content,
	}, nil
}
