package internal

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/cli/internal/log"
)

const chatSystemMessage = `You are a helpful and knowledgeable assistant.
Your job is to provide accurate and concise answers to general questions.
Be polite, clear, and informative in your responses, maintaining a friendly tone.
`

type Chat struct {
	config *Config

	systemMessage string
}

type ChatMessage struct {
	Content string
}

func NewChat(cfg *Config) (*Chat, error) {
	chat := Chat{
		config:        cfg,
		systemMessage: chatSystemMessage,
	}
	return &chat, nil
}

func (r *Chat) Send(ctx context.Context, input string) (*ChatMessage, error) {
	systemMessage := r.systemMessage
	userMessage := input

	log.Debugln(">>>SYSTEM:\n", systemMessage)
	log.Debugln(">>>USER:\n", userMessage)

	//
	model := r.config.Model

	client := openai.NewClient(
		option.WithAPIKey(r.config.ApiKey),
		option.WithBaseURL(r.config.BaseUrl),
	)

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemMessage),
			openai.UserMessage(userMessage),
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
		// dry-run mode
		if r.config.DryRunFile == "" {
			content = "Fake data!"
		} else {
			var err error
			content, err = ReadFile(r.config.DryRunFile)
			if err != nil {
				return nil, err
			}
		}
	}
	log.Debugf("<<<OPENAI:\nmodel: %s, content length: %v\n\n", model, len(content))

	return &ChatMessage{Content: content}, nil
}
