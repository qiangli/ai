package internal

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
)

type AskAgent struct {
	config *Config

	Role    string
	Message string

	autoMessage string
}

func NewAskAgent(cfg *Config, role, content string) (*AskAgent, error) {
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
		prompt, err := r.GeneratePrompt(ctx, r.Role, message, input)
		if err != nil {
			return nil, err
		}
		agent = prompt.Service
		message = prompt.RolePrompt
	}

	content, err := r.send(ctx, r.Role, message, input)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   agent,
		Content: content,
	}, nil
}

type AskAgentPrompt struct {
	Service    string `json:"service"`
	RolePrompt string `json:"agent_role_prompt"`
}

func (r *AskAgent) GeneratePrompt(ctx context.Context, role, prompt, input string) (*AskAgentPrompt, error) {
	content, err := r.send(ctx, role, prompt, input)
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

func (r *AskAgent) send(ctx context.Context, role, prompt, input string) (string, error) {
	roleMessage := buildRoleMessage(role, prompt)
	userMessage := buildRoleMessage("user", input)

	log.Debugf(">>>%s:\n%+v\n", strings.ToUpper(role), roleMessage)
	log.Debugf(">>>USER:\n%+v\n", userMessage)

	//
	model := r.config.Model

	client := openai.NewClient(
		option.WithAPIKey(r.config.ApiKey),
		option.WithBaseURL(r.config.BaseUrl),
		option.WithMiddleware(logMiddleware()),
	)

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			roleMessage,
			userMessage,
		}),
		Seed:  openai.Int(0),
		Model: openai.F(model),
	}

	var content string

	if !r.config.DryRun {
		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			return "", err
		}
		content = completion.Choices[0].Message.Content
	} else {
		content = r.config.DryRunContent
	}
	log.Debugf("<<<OPENAI:\nmodel: %s, content length: %v\n\n", model, len(content))
	return content, nil
}

const techAgentDummyResponse = `
{
 "service": "🖥️ Technical Support Agent",
 "agent_role_prompt": "You are a proficient AI technical support assistant. Your primary function is to provide detailed, accurate, and user-friendly instructions for troubleshooting and maintaining computer systems, particularly in Windows operating systems."
}
`
