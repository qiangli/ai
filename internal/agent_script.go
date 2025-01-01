package internal

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/tool"
	"github.com/qiangli/ai/internal/util"
)

type ScriptAgent struct {
	config *Config

	Role    string
	Message string
}

type ScriptAgentMessage struct {
	Content string
}

func NewScriptAgent(cfg *Config, role, content string) (*ScriptAgent, error) {
	if role == "" {
		role = string(openai.ChatCompletionMessageParamRoleSystem)
	}
	info, err := util.CollectSystemInfo()
	if err != nil {
		return nil, err
	}
	if content == "" {
		systemMessage, err := resource.GetSystemRoleContent(info)
		if err != nil {
			return nil, err
		}
		content = systemMessage
	}

	chat := ScriptAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &chat, nil
}

func (r *ScriptAgent) Send(ctx context.Context, command, input string) (*ScriptAgentMessage, error) {
	roleMessage := buildRoleMessage(r.Role, r.Message)
	userContent, err := resource.GetUserRoleContent(
		command, input,
	)
	if err != nil {
		return nil, err
	}
	userMessage := buildRoleMessage("user", userContent)

	log.Debugf(">>>%s:\n%+v\n", strings.ToUpper(r.Role), roleMessage)
	log.Debugf(">>>USER:\n%+v\n", userMessage)

	//
	tools := tool.Tools
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

				log.Debugf("\n\n*** tool call: %s\nprops: %+v\n", name, props)
				out, err := tool.RunTool(name, props)
				if err != nil {
					return nil, err
				}
				log.Debugf("\n*** tool call: %s\n%s\n", name, out)
				params.Messages.Value = append(params.Messages.Value, openai.ToolMessage(toolCall.ID, out))
			}
		}
	} else {
		// dry-run mode
		content = r.config.DryRunContent
	}

	log.Debugf("<<<OPENAI:\nmodel: %s, content length: %v\n\n", model, len(content))

	return &ScriptAgentMessage{Content: content}, nil
}
