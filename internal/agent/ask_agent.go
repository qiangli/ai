package agent

import (
	"context"
	"encoding/json"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

type AskAgent struct {
	config *llm.Config

	Role    string
	Message string

	autoMessage string
}

func NewAskAgent(cfg *llm.Config, role, content string) (*AskAgent, error) {
	if role == "" {
		role = "system"
	}
	autoMessage := resource.GetMetaRoleContent()

	agent := AskAgent{
		config:      cfg,
		Role:        role,
		Message:     content,
		autoMessage: autoMessage,
	}
	return &agent, nil
}

func (r *AskAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	var agent = "ASK"
	var message = r.Message

	if r.config.MetaPrompt {
		if message == "" {
			message = r.autoMessage
		}
		prompt, err := r.GeneratePrompt(r.config, ctx, r.Role, message, input)
		if err != nil {
			return nil, err
		}
		agent = prompt.Service
		message = prompt.RolePrompt
	}

	resp, err := llm.Chat(ctx, &llm.Message{
		Role:    r.Role,
		Prompt:  message,
		Model:   llm.Level1(r.config),
		Input:   input,
		DBCreds: r.config.DBConfig,
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
	RolePrompt string `json:"agent_role_prompt"`
}

func (r *AskAgent) GeneratePrompt(cfg *llm.Config, ctx context.Context, role, prompt, input string) (*AskAgentPrompt, error) {
	content, err := llm.Send(cfg, ctx, role, prompt, input)
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

const techAgentDummyResponse = `
{
 "service": "🖥️ Technical Support Agent",
 "agent_role_prompt": "You are a proficient AI technical support assistant. Your primary function is to provide detailed, accurate, and user-friendly instructions for troubleshooting and maintaining computer systems, particularly in Windows operating systems."
}
`
