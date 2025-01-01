package internal

import (
	"context"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/internal/log"
)

const chatSystemMessage = `You are a helpful and knowledgeable assistant.
Your job is to provide accurate and concise answers to general questions.
Be polite, clear, and informative in your responses, maintaining a friendly tone.
`

type Chat struct {
	config *Config

	Role    string
	Message string
}

type ChatMessage struct {
	Content string
}

func NewChat(cfg *Config, role, content string) (*Chat, error) {
	if role == "" {
		role = string(openai.ChatCompletionMessageParamRoleSystem)
	}
	if content == "" {
		content = chatSystemMessage
	}

	chat := Chat{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &chat, nil
}

func (r *Chat) Send(ctx context.Context, input string) (*ChatMessage, error) {
	roleMessage := buildRoleMessage(r.Role, r.Message)
	userMessage := buildRoleMessage("user", input)

	log.Debugf(">>>%s:\n%+v\n", strings.ToUpper(r.Role), roleMessage)
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
			return nil, err
		}
		content = completion.Choices[0].Message.Content
	} else {
		content = r.config.DryRunContent
	}
	log.Debugf("<<<OPENAI:\nmodel: %s, content length: %v\n\n", model, len(content))

	return &ChatMessage{Content: content}, nil
}
