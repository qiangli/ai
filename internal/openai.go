package internal

import (
	"context"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/internal/log"
)

func SendMessage(cfg *Config, ctx context.Context, role, prompt, input string) (string, error) {

	roleMessage := buildRoleMessage(role, prompt)
	userMessage := buildRoleMessage("user", input)

	log.Debugf(">>>%s:\n%+v\n", strings.ToUpper(role), roleMessage)
	log.Debugf(">>>USER:\n%+v\n", userMessage)

	//
	model := cfg.Model

	client := openai.NewClient(
		option.WithAPIKey(cfg.ApiKey),
		option.WithBaseURL(cfg.BaseUrl),
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

	if !cfg.DryRun {
		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			return "", err
		}
		content = completion.Choices[0].Message.Content
	} else {
		content = cfg.DryRunContent
	}
	log.Debugf("<<<OPENAI:\nmodel: %s, content length: %v\n\n", model, len(content))
	return content, nil
}
