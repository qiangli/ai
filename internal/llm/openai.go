package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/internal/log"
	// "github.com/qiangli/ai/internal/tool"
)

func Send(cfg *Config, ctx context.Context, role, prompt, input string) (string, error) {
	req := &Message{
		Role:    role,
		Prompt:  prompt,
		Model:   CreateModel(cfg),
		Input:   input,
		DBCreds: cfg.DBConfig,
	}

	resp, err := Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func Chat(ctx context.Context, req *Message) (*Message, error) {

	roleMessage := buildRoleMessage(req.Role, req.Prompt)
	userMessage := buildRoleMessage("user", req.Input)

	log.Debugf(">>>%s:\n%+v\n", strings.ToUpper(req.Role), roleMessage)
	log.Debugf(">>>USER:\n%+v\n", userMessage)

	model := req.Model

	client := openai.NewClient(
		option.WithAPIKey(model.ApiKey),
		option.WithBaseURL(model.BaseUrl),
		option.WithMiddleware(log.Middleware()),
	)

	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			roleMessage,
			userMessage,
		}),
		Seed:  openai.Int(0),
		Model: openai.F(model.Name),
	}

	if len(model.Tools) > 0 {
		params.Tools = openai.F([]openai.ChatCompletionToolParam(model.Tools))
	}

	resp := &Message{}
	// TODO
	var max = len(model.Tools) + 1

	if !model.DryRun {
		for tries := 0; tries < max; tries++ {
			log.Debugf("*** tries ***: %v\n", tries)

			completion, err := client.Chat.Completions.New(ctx, params)
			if err != nil {
				return nil, err
			}

			toolCalls := completion.Choices[0].Message.ToolCalls

			if len(toolCalls) == 0 {
				resp.Content = completion.Choices[0].Message.Content
				break
			}

			params.Messages.Value = append(params.Messages.Value, completion.Choices[0].Message)

			for _, toolCall := range toolCalls {
				var name = toolCall.Function.Name
				var props map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &props); err != nil {
					return nil, err
				}

				log.Debugf("\n\n*** tool call: %s props: %+v\n", name, props)
				toolCfg := &ToolConfig{
					Model:    model,
					DBConfig: req.DBCreds,
				}
				out, err := RunTool(toolCfg, ctx, name, props)
				if err != nil {
					return nil, err
				}
				log.Debugf("\n*** tool call: %s out: %s\n", name, out)
				params.Messages.Value = append(params.Messages.Value, openai.ToolMessage(toolCall.ID, out))
			}
		}
	} else {
		resp.Content = model.DryRunContent
	}

	log.Debugf("<<<OPENAI:\nmodel: %s, content length: %v\n\n", model, len(resp.Content))
	return resp, nil
}

// https://platform.openai.com/docs/guides/text-generation#developer-messages
func buildRoleMessage(role string, content string) openai.ChatCompletionMessageParamUnion {
	switch role {
	case "system":
		return openai.SystemMessage(content)
	case "assistant":
		return openai.AssistantMessage(content)
	case "user":
		return openai.UserMessage(content)
	// case "tool":
	// 	return openai.ToolMessage("", content)
	// case "function":
	// 	return openai.FunctionMessage("", content)
	case "developer":
		// return DeveloperMessage(content)
		return openai.SystemMessage(content)
	default:
		return nil
	}
}

func DeveloperMessage(content string) openai.ChatCompletionMessageParamUnion {
	return openai.ChatCompletionDeveloperMessageParam{
		Role: openai.F(openai.ChatCompletionDeveloperMessageParamRoleDeveloper),
		Content: openai.F([]openai.ChatCompletionContentPartTextParam{
			openai.TextPart(content),
		}),
	}
}
