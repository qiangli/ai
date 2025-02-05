package agent

import (
	"context"
	"encoding/json"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

type AskAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string

	autoMessage string
}

func NewAskAgent(cfg *internal.AppConfig) (*AskAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt
	if role == "" {
		role = "system"
	}
	autoMessage := resource.GetMetaRoleContent()

	agent := AskAgent{
		config:      cfg,
		Role:        role,
		Prompt:      prompt,
		autoMessage: autoMessage,
	}
	return &agent, nil
}

func (r *AskAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	var agent = "ASK"
	var message = r.Prompt
	var input = in.Input()

	if r.config.LLM.MetaPrompt {
		if message == "" {
			message = r.autoMessage
		}
		prompt, err := r.GeneratePrompt(ctx, r.Role, message, in.Intent())
		if err != nil {
			return nil, err
		}
		agent = prompt.Service
		message = prompt.RolePrompt
	}

	resp, err := llm.Chat(ctx, &internal.Message{
		Role:   r.Role,
		Prompt: message,
		Model:  internal.Level2(r.config.LLM),
		Input:  input,
	})
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   agent,
		Content: resp.Content,
	}, nil
}

type AskAgentPrompt struct {
	Service    string `json:"service"`
	RolePrompt string `json:"system_role_prompt"`
}

func (r *AskAgent) GeneratePrompt(ctx context.Context, role, prompt, input string) (*AskAgentPrompt, error) {
	model := internal.Level1(r.config.LLM)
	content, err := llm.Send(ctx, role, prompt, model, input)
	if err != nil {
		return nil, err
	}

	var resp AskAgentPrompt
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		// fallback instead of error
		return &AskAgentPrompt{
			Service:    "ASK",
			RolePrompt: "",
		}, nil
	}
	return &resp, nil
}
