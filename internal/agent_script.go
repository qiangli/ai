package internal

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/cli/internal/log"
	"github.com/qiangli/ai/cli/internal/tool"
)

type ScriptAgent struct {
	config *Config

	systemMessage string
}

type ScriptAgentMessage struct {
	Content string
}

func NewScriptAgent(cfg *Config) (*ScriptAgent, error) {
	systemMessage, err := GetSystemRoleMessage()
	if err != nil {
		return nil, err
	}

	chat := ScriptAgent{
		config:        cfg,
		systemMessage: systemMessage,
	}
	return &chat, nil
}

func (r *ScriptAgent) Send(ctx context.Context, command, message string) (*ScriptAgentMessage, error) {
	systemMessage := r.systemMessage

	userMessage, err := GetUserRoleMessage(
		command, message,
	)
	if err != nil {
		return nil, err
	}

	log.Debugln(">>>SYSTEM:\n", systemMessage)
	log.Debugln(">>>USER:\n", userMessage)

	//
	tools := tool.Tools
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
		Tools: openai.F(tools),
		Seed:  openai.Int(0),
		Model: openai.F(model),
	}

	var content string
	var max = 5

	if !r.config.DryRun {
		for tries := 0; tries < max; tries++ {
			log.Debugf("*** tries ***: %v\n", tries)

			completion, err := client.Chat.Completions.New(ctx, params)
			if err != nil {
				return nil, err
			}

			toolCalls := completion.Choices[0].Message.ToolCalls

			if len(toolCalls) == 0 {
				content = completion.Choices[0].Message.Content
				break
			}

			params.Messages.Value = append(params.Messages.Value, completion.Choices[0].Message)

			for _, toolCall := range toolCalls {
				var name = toolCall.Function.Name
				var props map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &props); err != nil {
					return nil, err
				}

				log.Debugf("tool call name: %s, props: %+v\n", name, props)
				out, err := tool.RunTool(name, props)
				if err != nil {
					return nil, err
				}
				log.Debugf("tool call name: %s, out:\n%s\n", name, out)
				params.Messages.Value = append(params.Messages.Value, openai.ToolMessage(toolCall.ID, out))
			}
		}
	} else {
		// dry-run mode
		if r.config.DryRunFile == "" {
			content = "Fake data!"
		} else {
			content, err = ReadFile(r.config.DryRunFile)
			if err != nil {
				return nil, err
			}
		}
	}

	log.Debugf("<<<OPENAI:\nmodel: %s, content length: %v\n\n", model, len(content))

	return &ScriptAgentMessage{Content: content}, nil
}
