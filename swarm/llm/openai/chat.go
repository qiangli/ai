package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/api/model"
	"github.com/qiangli/ai/swarm/middleware"
)

// https://platform.openai.com/docs/models

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

func NewClient(model *model.Model, vars *api.Vars) openai.Client {
	client := openai.NewClient(
		option.WithAPIKey(model.ApiKey),
		option.WithBaseURL(model.BaseUrl),
		option.WithMiddleware(middleware.Middleware(model, vars)),
	)
	return client
}

func Send(ctx context.Context, req *api.LLMRequest) (*api.LLMResponse, error) {
	log.Debugf(">>>OPENAI:\n req: %+v\n\n", req)

	var err error
	var resp *api.LLMResponse

	switch req.Model.Type {
	case model.OutputTypeImage:
		resp, err = generateImage(ctx, req)
	default:
		resp, err = call(ctx, req)
	}

	log.Debugf("<<<OPENAI:\n resp: %+v err: %v\n\n", resp, err)
	return resp, err
}

func call(ctx context.Context, req *api.LLMRequest) (*api.LLMResponse, error) {
	client := NewClient(req.Model, req.Vars)
	model := req.Model.Model

	params := openai.ChatCompletionNewParams{
		Seed:  openai.Int(0),
		Model: model,
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
			if v.ContentType != "" {
				messages = append(messages, openai.UserMessage(toContentPart(v.ContentType, []byte(v.Content))))
			} else {
				messages = append(messages, openai.UserMessage(v.Content))
			}
		// case "tool":
		// 	return openai.ToolMessage(content, id), nil
		// case "developer":
		// 	messages = append(messages, openai.DeveloperMessage(v.Content))
		default:
			log.Errorf("role not supported: %s", v.Role)
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

	log.Debugf("[OpenAI] params messages: %v tools: %v\n", len(params.Messages), len(params.Tools))

	for tries := range maxTurns {
		log.Infof("\033[33mⓄ\033[0m @%s [%v] %s %s\n", req.Agent, tries, model, req.Model.BaseUrl)

		log.Debugf("📡 *** sending request to %s ***: %v of %v\n%+v\n\n", req.Model.BaseUrl, tries, maxTurns, req)

		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			log.Errorf("\033[31m✗\033[0m %s\n", err)
			return nil, err
		}
		log.Infof("(%v)\n", completion.Choices[0].FinishReason)

		toolCalls := completion.Choices[0].Message.ToolCalls

		if len(toolCalls) == 0 {
			resp.Role = string(completion.Choices[0].Message.Role)
			// resp.ContentType = "text/plain"
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

			if out.MimeType != "" && !strings.HasPrefix(out.MimeType, "text/") {
				// TODO this is a hack and seems to work for non text parts
				// investigate this may fail for multi tool calls unless this is the last
				params.Messages = append(params.Messages, openai.ToolMessage(fmt.Sprintf("%s\nThe file content is included as data URL in the user message.", out.Message), toolCall.ID))
				params.Messages = append(params.Messages, openai.UserMessage(toContentPart(out.MimeType, []byte(out.Value))))
			} else {
				params.Messages = append(params.Messages, openai.ToolMessage(out.Value, toolCall.ID))
			}
		}
	}

	return resp, nil
}

// https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data
// data:[<media-type>][;base64],<data>
func dataURL(mime string, raw []byte) string {
	encoded := base64.StdEncoding.EncodeToString(raw)
	d := fmt.Sprintf("data:%s;base64,%s", mime, encoded)
	return d
}

func toContentPart(mimeType string, raw []byte) []openai.ChatCompletionContentPartUnionParam {
	// https://mimesniff.spec.whatwg.org/
	log.Debugf("[OpenAI] toContentPart: %s %v\n", mimeType, len(raw))
	switch {
	case strings.HasPrefix(mimeType, "text/"):
		return []openai.ChatCompletionContentPartUnionParam{
			openai.TextContentPart(string(raw)),
		}
	case strings.HasPrefix(mimeType, "image/"):
		return []openai.ChatCompletionContentPartUnionParam{
			openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
				URL: dataURL(mimeType, raw),
			}),
		}
	case strings.HasPrefix(mimeType, "audio/"):
		return []openai.ChatCompletionContentPartUnionParam{
			openai.InputAudioContentPart(openai.ChatCompletionContentPartInputAudioInputAudioParam{
				Data: dataURL(mimeType, raw),
			}),
		}
	default:
		return []openai.ChatCompletionContentPartUnionParam{
			openai.FileContentPart(openai.ChatCompletionContentPartFileFileParam{
				FileData: param.NewOpt(dataURL(mimeType, raw)),
			}),
		}
	}
}

// func NewContentPart(filePath string) ([]openai.ChatCompletionContentPartUnionParam, error) {
// 	raw, err := os.ReadFile(filePath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	mimeType := http.DetectContentType(raw)
// 	return toContentPart(mimeType, raw), nil
// }

func generateImage(ctx context.Context, req *api.LLMRequest) (*api.LLMResponse, error) {
	messages := make([]string, 0)
	for _, v := range req.Messages {
		messages = append(messages, v.Content)
	}

	client := NewClient(req.Model, req.Vars)
	prompt := strings.Join(messages, "\n")
	model := req.Model.Model

	resp := &api.LLMResponse{
		ContentType: api.ContentTypeB64JSON,
	}

	log.Infof("@%s %s %s\n", req.Agent, req.Model, req.Model.BaseUrl)

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

	if q, ok := qualityMap[req.Vars.Extra["quality"]]; ok {
		imageQuality = q
	}
	if s, ok := sizeMap[req.Vars.Extra["size"]]; ok {
		imageSize = s
	}
	if s, ok := styleMap[req.Vars.Extra["style"]]; ok {
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
