package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
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

func Send(ctx context.Context, req *api.Request) (*api.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI:\n req: %+v\n", req)

	var err error
	var resp *api.Response

	resp, err = call(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI:\n resp: %+v err: %v\n", resp, err)
	return resp, err
}

func call(ctx context.Context, req *api.Request) (*api.Response, error) {
	client, err := NewClient(req.Model, req.Token())
	if err != nil {
		return nil, err
	}

	var params = openai.ChatCompletionNewParams{
		Model: req.Model.Model,
	}
	if req.Arguments != nil {
		setChatCompletionNewParams(&params, req.Arguments)
	}
	if len(req.Messages) > 0 {
		var messages []openai.ChatCompletionMessageParamUnion
		for _, v := range req.Messages {
			// https://platform.openai.com/docs/guides/text-generation#developer-messages
			// https://model-spec.openai.com/2025-02-12.html
			switch v.Role {
			case "system":
				messages = append(messages, openai.SystemMessage(v.Content))
			case "developer":
				messages = append(messages, openai.DeveloperMessage(v.Content))
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
			default:
				// log.GetLogger(ctx).Errorf("Role not supported: %s", v.Role)
			}
		}
		params.Messages = messages
	}

	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("no input message")
	}

	if len(req.Tools) > 0 {
		var tools []openai.ChatCompletionToolUnionParam
		for _, f := range req.Tools {
			tools = append(tools, defineTool(f.ID(), f.Description, f.Parameters))
		}
		params.Tools = tools
	}

	var maxTurns = req.MaxTurns()
	if maxTurns <= 0 {
		maxTurns = 3
	}

	var resp = &api.Response{}

	log.GetLogger(ctx).Debugf("[OpenAI] params messages: %v tools: %v\n", len(params.Messages), len(params.Tools))

	for tries := range maxTurns {
		log.GetLogger(ctx).Infof("Ⓞ @%s [%v/%v] chat %s/%s\n", req.Name, tries, maxTurns, req.Model.Provider, req.Model.Model)

		log.GetLogger(ctx).Debugf("📡 sending chat request to %s: %v of %v\n%+v\n", req.Model.BaseUrl, tries, maxTurns, req)

		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			log.GetLogger(ctx).Errorf("❌ %s\n", err)
			return nil, err
		}
		log.GetLogger(ctx).Infof("(%s)\n", formatReason(completion.Choices[0].FinishReason))

		toolCalls := completion.Choices[0].Message.ToolCalls
		if len(toolCalls) == 0 {
			resp.Result = &api.Result{
				Role:     string(completion.Choices[0].Message.Role),
				MimeType: "text/plain",
				Value:    completion.Choices[0].Message.Content,
			}

			// request completed
			return resp, nil
		}

		params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())
		// results := runTools(ctx, req.RunTool, toolCalls, maxThreadLimit)
		calls := make([]*api.ToolCall, len(toolCalls))
		for i, v := range toolCalls {
			var props map[string]any
			if err := json.Unmarshal([]byte(v.Function.Arguments), &props); err != nil {
				return nil, err
			}
			// // copy request args if not found in props
			// for k, v := range req.Arguments {
			// 	if _, ok := props[k]; !ok {
			// 		props[k] = v
			// 	}
			// }

			calls[i] = api.NewToolCall(
				v.ID,
				v.Function.Name,
				props,
			)
		}
		results := runToolsV3(ctx, req.Runner, calls, maxThreadLimit)
		for i, out := range results {
			if out == nil {
				params.Messages = append(params.Messages, openai.ToolMessage("no result", calls[i].ID))
				continue
			}
			// if out.State == api.StateExit {
			// 	resp.Result = out
			// 	return resp, nil
			// }
			if out.State == api.StateTransfer {
				resp.Result = out
				return resp, nil
			}

			params.Messages = append(params.Messages, openai.ToolMessage(out.Value, calls[i].ID))
		}
	}

	// not finished due the max turns reached
	if resp.Result == nil {
		return nil, fmt.Errorf("Empty response. Max turns reached: %v. Try again with a higher value for the 'max_turns' parameter", maxTurns)
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

// https://platform.openai.com/docs/api-reference/responses/create
func setChatCompletionNewParams(params *openai.ChatCompletionNewParams, args api.Arguments) {
	// Number between -2.0 and 2.0. Positive values penalize new tokens based on their
	// existing frequency in the text so far, decreasing the model's likelihood to
	// repeat the same line verbatim.
	if v, ok := args.Get2("frequency_penalty"); ok {
		params.FrequencyPenalty = openai.Float(toFloat64(v, 0.0))
	}

	// Whether to return log probabilities of the output tokens or not. If true,
	// returns the log probabilities of each output token returned in the `content` of
	// `message`.
	if v, ok := args.Get2("logprobs"); ok {
		params.Logprobs = openai.Bool(toBool(v, false))
	}

	// An upper bound for the number of tokens that can be generated for a completion,
	// including visible output tokens and reasoning tokens.
	if v, ok := args.Get2("max_completion_tokens"); ok {
		params.MaxCompletionTokens = openai.Int(toInt64(v, 512))
	}

	// The maximum number of tokens that can be generated in the chat completion.
	// if v, ok := args["max_tokens"]; ok {
	// 	params.MaxTokens = openai.Int(toInt64(v, 512))
	// }

	// // How many chat completion choices to generate for each input message.
	// if v, ok := args.Get2("n"); ok {
	// 	params.N = openai.Int(toInt64(v, 1))
	// }
	// only 1
	params.N = openai.Int(1)

	// Number between -2.0 and 2.0. Positive values penalize new tokens based on
	// whether they appear in the text so far, increasing the model's likelihood to
	// talk about new topics.
	if v, ok := args.Get2("presence_penalty"); ok {
		params.PresencePenalty = openai.Float(toFloat64(v, 0.0))
	}

	// If specified, the system will make a best effort to sample deterministically.
	if v, ok := args.Get2("seed"); ok {
		params.Seed = openai.Int(toInt64(v, 0))
	}

	// Whether or not to store the output of this chat completion request.
	if v, ok := args.Get2("store"); ok {
		params.Store = openai.Bool(toBool(v, false))
	}

	// What sampling temperature to use, between 0 and 2.
	if v, ok := args.Get2("temperature"); ok {
		params.Temperature = openai.Float(toFloat64(v, 1.0))
	}

	// An integer specifying the number of most likely tokens to return.
	if v, ok := args.Get2("top_logprobs"); ok {
		params.TopLogprobs = openai.Int(toInt64(v, 0))
	}

	// An alternative to sampling with temperature, called nucleus sampling.
	if v, ok := args.Get2("top_p"); ok {
		params.TopP = openai.Float(toFloat64(v, 1.0))
	}

	// Whether to enable parallel function calling during tool use.
	if v, ok := args.Get2("parallel_tool_calls"); ok {
		params.ParallelToolCalls = openai.Bool(toBool(v, false))
	}
}
