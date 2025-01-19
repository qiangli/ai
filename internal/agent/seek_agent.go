package agent

import (
	"context"

	"github.com/qiangli/ai/internal"
)

type SeekAgent struct {
	config *internal.LLMConfig

	Role    string
	Message string
}

func NewSeekAgent(cfg *internal.LLMConfig, role, content string) (*SeekAgent, error) {
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
