package internal

import (
	"context"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/internal/log"
)

type ChatAgent struct {
	config *Config

	Role    string
	Message string
}

func NewChatAgent(cfg *Config, role, content string) (*ChatAgent, error) {
	if role == "" {
		role = "system"
	}

	chat := ChatAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &chat, nil
}

func (r *ChatAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	var message = r.Message

	content, err := r.send(ctx, r.Role, message, input)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "CHAT",
		Content: content,
	}, nil
}

func (r *ChatAgent) send(ctx context.Context, role, prompt, input string) (string, error) {

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
