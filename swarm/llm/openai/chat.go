package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/middleware"
)

// https://platform.openai.com/docs/models

// https://github.com/openai/openai-go/tree/main/examples
func defineTool(name, description string, parameters map[string]any) openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionToolUnionParam{
		OfFunction: &openai.ChatCompletionFunctionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        name,
				Description: openai.String(description),
				Parameters:  openai.FunctionParameters(parameters),
			},
		},
	}
}

func NewClient(model *api.Model, vars *api.Vars) (*openai.Client, error) {
	client := openai.NewClient(
		option.WithAPIKey(model.ApiKey),
		option.WithBaseURL(model.BaseUrl),
		option.WithMiddleware(middleware.Middleware(model, vars)),
	)
	return &client, nil
}

func Send(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI:\n req: %+v\n", req)

	var err error
	var resp *llm.Response

	resp, err = call(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI:\n resp: %+v err: %v\n", resp, err)
	return resp, err
}

func call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	client, err := NewClient(req.Model, req.Vars)
	if err != nil {
		return nil, err
	}
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
			log.GetLogger(ctx).Errorf("Role not supported: %s", v.Role)
		}
	}
	params.Messages = messages

	if len(req.Tools) > 0 {
		var tools []openai.ChatCompletionToolUnionParam
		for _, f := range req.Tools {

			tools = append(tools, defineTool(f.ID(), f.Description, f.Parameters))
		}

		params.Tools = tools
	}

	maxTurns := req.MaxTurns
	if maxTurns == 0 {
		maxTurns = 1
	}
	resp := &llm.Response{}

	log.GetLogger(ctx).Debugf("[OpenAI] params messages: %v tools: %v\n", len(params.Messages), len(params.Tools))

	for tries := range maxTurns {
		log.GetLogger(ctx).Infof("Ⓞ @%s [%v] %s %s\n", req.Agent, tries, model, req.Model.BaseUrl)

		log.GetLogger(ctx).Debugf("📡 sending request to %s: %v of %v\n%+v\n", req.Model.BaseUrl, tries, maxTurns, req)

		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			log.GetLogger(ctx).Errorf("❌ %s\n", err)
			return nil, err
		}
		log.GetLogger(ctx).Infof("(%v)\n", completion.Choices[0].FinishReason)

		toolCalls := completion.Choices[0].Message.ToolCalls

		if len(toolCalls) == 0 {
			resp.Role = string(completion.Choices[0].Message.Role)
			// resp.ContentType = "text/plain"
			// resp.Content = completion.Choices[0].Message.Content
			resp.Result = &api.Result{
				MimeType: "text/plain",
				Value:    completion.Choices[0].Message.Content,
			}
			break
		}

		params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())

		for i, toolCall := range toolCalls {
			var name = toolCall.Function.Name
			var props map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &props); err != nil {
				return nil, err
			}

			log.GetLogger(ctx).Debugf("\n* tool call: %v %s props: %+v\n", i, name, props)

			//
			out, err := req.RunTool(ctx, name, props)
			if err != nil {
				out = &api.Result{
					Value: fmt.Sprintf("%s", err),
				}
			}

			log.GetLogger(ctx).Debugf("* tool call: %s out: %+v\n", name, out)
			resp.Result = out

			if out.State == api.StateExit {
				// resp.Content = out.Value
				return resp, nil
			}
			if out.State == api.StateTransfer {
				return resp, nil
			}

			// if out.MimeType != "" && !strings.HasPrefix(out.MimeType, "text/") {
			// 	// TODO this is a hack and seems to work for non text parts
			// 	// investigate this may fail for multi tool calls unless this is the last
			// 	// params.Messages = append(params.Messages, openai.ToolMessage(fmt.Sprintf("%s\nThe file content is included as data URL in the user message.", out.Message), toolCall.ID))
			// 	// params.Messages = append(params.Messages, openai.UserMessage(toContentPart(out.MimeType, []byte(out.Value))))
			// } else {
			// 	params.Messages = append(params.Messages, openai.ToolMessage(out.Value, toolCall.ID))
			// }
			params.Messages = append(params.Messages, openai.ToolMessage(out.Value, toolCall.ID))
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
