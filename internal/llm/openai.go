package llm

import (
	"context"
	"encoding/json"
	"fmt"
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

func RunTool(cfg *internal.ToolConfig, ctx context.Context, name string, props map[string]interface{}) (string, error) {
	panic("No longer supported with Chat, migrate to enw Call")
}

func NewClient(apiKey, baseUrl string) *openai.Client {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseUrl),
		option.WithMiddleware(log.Middleware(internal.DryRun, internal.DryRunContent)),
	)
	return client
}

func Send(ctx context.Context, role, prompt string, model *internal.Model, input string) (string, error) {
	req := &internal.Message{
		Role:   role,
		Prompt: prompt,
		Model:  model,
		Input:  input,
	}

	resp, err := Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func Chat(ctx context.Context, req *internal.Message) (*internal.Message, error) {

	roleMessage := buildRoleMessage(req.Role, req.Prompt)
	userMessage := buildRoleMessage("user", req.Input)

	log.Debugf(">>>%s:\n%+v\n", strings.ToUpper(req.Role), roleMessage)
	log.Debugf(">>>USER:\n%+v\n", userMessage)

	model := req.Model

	client := NewClient(model.ApiKey, model.BaseUrl)

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

	resp := &internal.Message{}
	// TODO
	var max = len(model.Tools) + 1

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

			log.Debugf("\n\n>>> tool call: %s props: %+v\n", name, props)
			toolCfg := &internal.ToolConfig{
				Model:    model,
				DBConfig: req.DBCreds,
				Next:     req.Next,
			}
			out, err := RunTool(toolCfg, ctx, name, props)
			if err != nil {
				return nil, err
			}
			log.Debugf("\n<<< tool call: %s out: %s\n", name, out)
			params.Messages.Value = append(params.Messages.Value, openai.ToolMessage(toolCall.ID, out))
		}
	}

	log.Debugf("<<<OPENAI:\nmodel: %s, content length: %v\n\n", model, len(resp.Content))
	return resp, nil
}

func GenerateImage(ctx context.Context, req *internal.Message) (*internal.Message, error) {
	roleMessage := buildRoleMessage(req.Role, req.Prompt)
	userMessage := buildRoleMessage("user", req.Input)

	log.Debugf(">>>%s:\n%+v\n", strings.ToUpper(req.Role), roleMessage)
	log.Debugf(">>>USER:\n%+v\n", userMessage)

	prompt := fmt.Sprintf("%s\n===%s", roleMessage, userMessage)

	model := req.Model

	client := NewClient(model.ApiKey, model.BaseUrl)

	resp := &internal.Message{}

	// Base64
	image, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         openai.String(prompt),
		Model:          openai.F(model.Name),
		ResponseFormat: openai.F(openai.ImageGenerateParamsResponseFormatB64JSON),
		N:              openai.Int(1),
	})
	if err != nil {
		return nil, err
	}

	resp.Content = image.Data[0].B64JSON

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

type Request struct {
	BaseUrl string
	ApiKey  string
	Model   string

	// History  []*Message
	Messages []*Message

	MaxTurns int
	RunTool  func(ctx context.Context, name string, props map[string]any) (*api.Result, error)

	Tools []*ToolFunc
}

type ToolCall = openai.ChatCompletionMessageToolCall

type Message struct {
	Role    string
	Content string
	Sender  string

	ToolCalls []ToolCall
}

type ToolFunc struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

type Response struct {
	Role    string
	Content string

	Transfer  bool
	NextAgent string
}

func Call(ctx context.Context, req *Request) (*Response, error) {
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

	resp := &Response{}

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

			if out.State == api.StateExit {
				return &Response{
					Content: out.Value,
				}, nil
			}
			if out.State == api.StateTransfer {
				return &Response{
					Transfer:  true,
					NextAgent: out.NextAgent,
				}, nil
			}
			params.Messages.Value = append(params.Messages.Value, openai.ToolMessage(toolCall.ID, out.Value))
		}
	}

	return resp, nil
}
