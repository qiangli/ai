package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

// https://github.com/openai/openai-go/tree/main/examples
func defineTool(name, description string, parameters map[string]any) openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Function: openai.FunctionDefinitionParam{
			Name:        name,
			Description: openai.String(description),
			Parameters:  openai.FunctionParameters(parameters),
		},
	}
}

func NewClient(apiKey, baseUrl string) openai.Client {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseUrl),
		option.WithMiddleware(log.Middleware(internal.DryRun, internal.DryRunContent)),
	)
	return client
}

func Send(ctx context.Context, req *api.LLMRequest) (*api.LLMResponse, error) {
	log.Debugf(">>>OPENAI:\n Model type: %s Model: %s, Messages: %v Tools: %v\n\n", req.ModelType, req.Model, len(req.Messages), len(req.Tools))

	var err error
	var resp *api.LLMResponse

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

	log.Debugf("<<<OPENAI:\n Content type: %s Content: %v\n\n", resp.ContentType, len(resp.Content))
	return resp, nil
}

func call(ctx context.Context, req *api.LLMRequest) (*api.LLMResponse, error) {
	client := NewClient(req.ApiKey, req.BaseUrl)

	params := openai.ChatCompletionNewParams{
		Seed:  openai.Int(0),
		Model: req.Model,
	}

	var messages []openai.ChatCompletionMessageParamUnion
	for _, v := range req.Messages {
		// https://platform.openai.com/docs/guides/text-generation#developer-messages
		switch v.Role {
		case "system":
			messages = append(messages, openai.SystemMessage(v.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(v.Content))
		case "user":
			messages = append(messages, openai.UserMessage(v.Content))
		// case "tool":
		// 	return openai.ToolMessage(content, id), nil
		case "developer":
			messages = append(messages, openai.DeveloperMessage(v.Content))
		default:
			return nil, fmt.Errorf("role not supported: %s", v.Role)
		}
	}
	params.Messages = messages

	if len(req.Tools) > 0 {
		var tools []openai.ChatCompletionToolParam
		for _, f := range req.Tools {
			tools = append(tools, defineTool(f.ID(), f.Description, f.Parameters))
		}
		params.Tools = tools
	}

	maxTurns := req.MaxTurns
	if maxTurns == 0 {
		maxTurns = 1
	}
	resp := &api.LLMResponse{}

	for tries := range maxTurns {
		log.Infof("⚡ @%s [%v] %s %s\n", req.Agent, tries, req.Model, req.BaseUrl)

		log.Debugf("📡 *** sending request to %s ***: %v of %v\n%+v\n\n", req.BaseUrl, tries, maxTurns, req)

		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			log.Errorf("✗ %s\n", err)
			return nil, err
		}
		log.Infof("(%v)\n", completion.Choices[0].FinishReason)

		toolCalls := completion.Choices[0].Message.ToolCalls

		if len(toolCalls) == 0 {
			resp.Role = string(completion.Choices[0].Message.Role)
			resp.Content = completion.Choices[0].Message.Content
			break
		}

		params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())
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

			params.Messages = append(params.Messages, openai.ToolMessage(out.Value, toolCall.ID))
		}
	}

	return resp, nil
}

func generateImage(ctx context.Context, req *api.LLMRequest) (*api.LLMResponse, error) {
	messages := make([]string, 0)
	for _, v := range req.Messages {
		messages = append(messages, v.Content)
	}

	client := NewClient(req.ApiKey, req.BaseUrl)
	prompt := strings.Join(messages, "\n")
	model := req.Model

	resp := &api.LLMResponse{
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
		Prompt:         prompt,
		Model:          model,
		ResponseFormat: imageFormat,
		Quality:        imageQuality,
		Size:           imageSize,
		Style:          imageStyle,
		N:              openai.Int(1),
	})
	if err != nil {
		return nil, err
	}
	log.Infof("✨ %v %v %v\n", imageQuality, imageSize, imageStyle)

	resp.Content = image.Data[0].B64JSON

	return resp, nil
}
