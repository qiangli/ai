package agent

import (
	"context"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

// HelpAgent auto selects the best agent to handle the user query
type HelpAgent struct {
	config *internal.AppConfig

	Role    string
	Message string
}

func NewHelpAgent(cfg *internal.AppConfig) (*HelpAgent, error) {
	role := cfg.Role
	content := cfg.Prompt

	if role == "" {
		role = "system"
	}
	if content == "" {
		content = resource.GetCliAgentDetectSystem()
	}

	cfg.LLM.Tools = llm.GetAIHelpTools()

	agent := HelpAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *HelpAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	var clip = in.Clip()

	content, err := llm.Send(r.config.LLM, ctx, r.Role, r.Message, clip)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "AI",
		Content: content,
	}, nil
}
