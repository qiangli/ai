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

func define(name, description string, parameters map[string]any) openai.ChatCompletionToolParam {
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
			tools = append(tools, define(f.ID(), f.Description, f.Parameters))
		}
		params.Tools = openai.F(tools)
	}

	maxTurns := req.MaxTurns
	if maxTurns == 0 {
		maxTurns = 3
	}
	resp := &api.Response{}

	for tries := range maxTurns {
		log.Debugf("*** sending request to %s ***: %v of %v\n", req.BaseUrl, tries, maxTurns)
		for _, v := range params.Messages.Value {
			log.Debugf(">>> message: %+v\n", v)
		}

		log.Infof("[%v] @%s %s %s\n", tries, req.Agent, req.Model, req.BaseUrl)

		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			log.Errorf("✗ %s\n", err)
			return nil, err
		}
		log.Infof("⣿ %v\n", completion.Choices[0].FinishReason)

		toolCalls := completion.Choices[0].Message.ToolCalls

		if len(toolCalls) == 0 {
			resp.Role = string(completion.Choices[0].Message.Role)
			resp.Content = completion.Choices[0].Message.Content
			break
		}

		params.Messages.Value = append(params.Messages.Value, completion.Choices[0].Message)

		for i, toolCall := range toolCalls {
			var name = toolCall.Function.Name
			var props map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &props); err != nil {
				return nil, err
			}

			log.Debugf("\n\n>>> tool call: %v %s props: %+v\n", i, name, props)

			//
			out, err := req.RunTool(ctx, name, props)
			if err != nil {
				out = &api.Result{
					Value: fmt.Sprintf("%s", err),
				}
			}

			log.Debugf("\n<<< tool call: %s out: %+v\n", name, out)
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

	log.Infof("@%s %s %s\n", req.Agent, req.Model, req.BaseUrl)

	var imageFormat = openai.ImageGenerateParamsResponseFormatB64JSON

	var qualityMap = map[string]openai.ImageGenerateParamsQuality{
		"standard": openai.ImageGenerateParamsQualityStandard,
		"hd":       openai.ImageGenerateParamsQualityHD,
	}
	var sizeMap = map[string]openai.ImageGenerateParamsSize{
		"256x256":   openai.ImageGenerateParamsSize256x256,
		"512x512":   openai.ImageGenerateParamsSize512x512,
		"1024x1024": openai.ImageGenerateParamsSize1024x1024,
		"1792x1024": openai.ImageGenerateParamsSize1792x1024,
		"1024x1792": openai.ImageGenerateParamsSize1024x1792,
	}
	var styleMap = map[string]openai.ImageGenerateParamsStyle{
		"vivid":   openai.ImageGenerateParamsStyleVivid,
		"natural": openai.ImageGenerateParamsStyleNatural,
	}

	var imageQuality = openai.ImageGenerateParamsQualityStandard
	var imageSize = openai.ImageGenerateParamsSize1024x1024
	var imageStyle = openai.ImageGenerateParamsStyleNatural
	if q, ok := qualityMap[req.ImageQuality]; ok {
		imageQuality = q
	}
	if s, ok := sizeMap[req.ImageSize]; ok {
		imageSize = s
	}
	if s, ok := styleMap[req.ImageStyle]; ok {
		imageStyle = s
	}

	image, err := client.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         openai.String(prompt),
		Model:          openai.F(model),
		ResponseFormat: openai.F(imageFormat),
		Quality:        openai.F(imageQuality),
		Size:           openai.F(imageSize),
		Style:          openai.F(imageStyle),
		N:              openai.Int(1),
	})
	if err != nil {
		return nil, err
	}
	log.Infof("✨ %v %v %v\n", imageQuality, imageSize, imageStyle)

	resp.Content = image.Data[0].B64JSON

	return resp, nil
}
