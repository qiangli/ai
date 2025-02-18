package llm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
)

// https://github.com/openai/openai-go/tree/main/examples

func define(name, description string, parameters map[string]interface{}) openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Type: openai.F(openai.ChatCompletionToolTypeFunction),
		Function: openai.F(openai.FunctionDefinitionParam{
			Name:        openai.String(name),
			Description: openai.String(description),
			Parameters:  openai.F(openai.FunctionParameters(parameters)),
		}),
	}
}

func NewClient(apiKey, baseUrl string) *openai.Client {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseUrl),
		option.WithMiddleware(log.Middleware(internal.DryRun, internal.DryRunContent)),
	)
	return client
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
	case "tool":
		return openai.ToolMessage("", content)
	// case "function":
	// 	return openai.FunctionMessage("", content)
	case "developer":
		// return DeveloperMessage(content)
		return openai.SystemMessage(content)
	default:
		return nil
	}
}

func buildMessage(id string, role string, content string) openai.ChatCompletionMessageParamUnion {
	switch role {
	case "system":
		return openai.SystemMessage(content)
	case "assistant":
		return openai.AssistantMessage(content)
	case "user":
		return openai.UserMessage(content)
	case "tool":
		return openai.ToolMessage(id, content)
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

func Send(ctx context.Context, req *api.Request) (*api.Response, error) {
	log.Debugf(">>>OPENAI:\n type: %s model: %s, messages: %v tools: %v\n\n", req.ModelType, req.Model, len(req.Messages), len(req.Tools))

	var err error
	var resp *api.Response

	switch req.ModelType {
	case api.ModelTypeImage:
		resp, err = generateImage(ctx, req)
	default:
		resp, err = call(ctx, req)
	}

	if err != nil {
		log.Errorf("***OPENAI: %s\n\n", err)
		return nil, err
	}

	log.Debugf("<<<OPENAI:\n type: %s transfer: %+v, content: %v\n\n", resp.ContentType, resp.Result, len(resp.Content))
	return resp, nil
}

func call(ctx context.Context, req *api.Request) (*api.Response, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0)
	for _, v := range req.Messages {
		msg := buildMessage("", v.Role, v.Content)
		messages = append(messages, msg)
	}

	client := NewClient(req.ApiKey, req.BaseUrl)

	params := openai.ChatCompletionNewParams{
		Messages: openai.F(messages),
		Seed:     openai.Int(0),
		Model:    openai.F(req.Model),
	}

	if len(req.Tools) > 0 {
		tools := make([]openai.ChatCompletionToolParam, 0)
		for _, f := range req.Tools {
			tools = append(tools, define(f.Name, f.Description, f.Parameters))
		}
		params.Tools = openai.F(tools)
	}

	resp := &api.Response{}

	for tries := 0; tries < req.MaxTurns; tries++ {
		log.Debugf("*** sending to %s ***: %v of %v\n", req.BaseUrl, tries, req.MaxTurns)

		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, err
		}

		toolCalls := completion.Choices[0].Message.ToolCalls

		if len(toolCalls) == 0 {
			resp.Role = string(completion.Choices[0].Message.Role)
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

			log.Debugf("\n\n>>> tool call: %s props: %+v\n", name, props)

			//
			out, err := req.RunTool(ctx, name, props)
			if err != nil {
				return nil, err
			}

			log.Debugf("\n<<< tool call: %s out: %s\n", name, out)
			resp.Result = out

			if out.State == api.StateExit {
				resp.Content = out.Value
				return resp, nil
			}
			if out.State == api.StateTransfer {
				return resp, nil
			}
			params.Messages.Value = append(params.Messages.Value, openai.ToolMessage(toolCall.ID, out.Value))
		}
	}

	return resp, nil
}

func generateImage(ctx context.Context, req *api.Request) (*api.Response, error) {
	messages := make([]string, 0)
	for _, v := range req.Messages {
		messages = append(messages, v.Content)
	}

	client := NewClient(req.ApiKey, req.BaseUrl)
	prompt := strings.Join(messages, "\n")
	model := req.Model

	resp := &api.Response{
		ContentType: api.ContentTypeB64JSON,
	}

	// Base64
	image, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         openai.String(prompt),
		Model:          openai.F(model),
		ResponseFormat: openai.F(openai.ImageGenerateParamsResponseFormatB64JSON),
		N:              openai.Int(1),
	})
	if err != nil {
		return nil, err
	}

	resp.Content = image.Data[0].B64JSON

	return resp, nil
}
