package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared/constant"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
)

func SendV3(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI v3:\n req: %+v\n", req)

	var err error
	var resp *llm.Response

	resp, err = respond(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI v3:\n resp: %+v err: %v\n", resp, err)
	return resp, err
}

func respond(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	client, err := NewClientV3(req.Model, req.Token(), req.Vars)
	if err != nil {
		return nil, err
	}

	var params = responses.ResponseNewParams{
		Model:           req.Model.Model,
		MaxOutputTokens: openai.Int(512),
	}
	if req.Arguments != nil {
		setResponseNewParams(&params, req.Arguments)
	}
	if len(req.Messages) > 0 {
		var instructions []string
		var messages []responses.ResponseInputItemUnionParam
		for _, v := range req.Messages {
			if len(v.Content) == 0 {
				return nil, fmt.Errorf("empty message")
			}
			switch v.Role {
			case "system", "developer":
				instructions = append(instructions, v.Content)
			// 	messages = append(messages, defineInputParam(v.Role, v.Content))
			case "assistant":
				messages = append(messages, defineAssistantInputParam(v.Role, v.Content))
			case "user":
				if v.ContentType != "" {
					messages = append(messages, defineFileInputParam(v.Role, "", v.Content))
				} else {
					messages = append(messages, defineUserInputParam(v.Role, v.Content))
				}
			default:
				// log.GetLogger(ctx).Errorf("Role not supported: %s", v.Role)
			}
		}
		if len(instructions) > 0 {
			params.Instructions = openai.String(strings.Join(instructions, "\n"))
		}
		params.Input = responses.ResponseNewParamsInputUnion{
			OfInputItemList: messages,
		}
	}
	if len(req.Tools) > 0 {
		var tools []responses.ToolUnionParam
		for _, f := range req.Tools {
			tools = append(tools, defineToolV3(f.ID(), f.Description, f.Parameters))
		}
		params.Tools = tools
	}

	var maxTurns = req.MaxTurns
	if maxTurns == 0 {
		maxTurns = 1
	}

	var resp = &llm.Response{}

	log.GetLogger(ctx).Debugf("[OpenAI] params messages: %v tools: %v\n", len(params.Input.OfInputItemList), len(params.Tools))

	for tries := range maxTurns {
		log.GetLogger(ctx).Infof("Ⓞ @%s [%v] %s/%s\n", req.Name, tries, req.Model.Provider, req.Model.Model)

		log.GetLogger(ctx).Debugf("📡 sending request to %s: %v of %v\n%+v\n", req.Model.BaseUrl, tries, maxTurns, req)

		//
		result, err := client.Responses.New(ctx, params)
		if err != nil {
			log.GetLogger(ctx).Errorf("❌ %s\n", err)
			return nil, err
		}
		log.GetLogger(ctx).Infof("(%v)\n", result.Status)

		params.PreviousResponseID = openai.String(result.ID)
		params.Input = responses.ResponseNewParamsInputUnion{}

		var calls []*api.ToolCall
		for _, output := range result.Output {
			if output.Type == "function_call" {
				v := output.AsFunctionCall()
				var props map[string]any
				if err := json.Unmarshal([]byte(v.Arguments), &props); err != nil {
					return nil, err
				}

				calls = append(calls, &api.ToolCall{
					ID:        v.CallID,
					Name:      v.Name,
					Arguments: props,
				})
			}
		}

		//
		if len(calls) == 0 {
			resp.Result = &api.Result{
				Value: result.OutputText(),
			}
			return resp, nil
		}

		results := runToolsV3(ctx, req.RunTool, calls, maxThreadLimit)
		for i, out := range results {
			if out == nil {
				params.Input.OfInputItemList = append(params.Input.OfInputItemList, responses.ResponseInputItemParamOfFunctionCallOutput(calls[i].ID, "no result"))
				continue
			}
			if out.State == api.StateExit {
				resp.Result = out
				return resp, nil
			}
			params.Input.OfInputItemList = append(params.Input.OfInputItemList, responses.ResponseInputItemParamOfFunctionCallOutput(calls[i].ID, out.Value))
		}
	}

	return resp, nil
}

func defineUserInputParam(role, text string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemParamOfMessage(
		responses.ResponseInputMessageContentListParam{
			responses.ResponseInputContentUnionParam{
				OfInputText: &responses.ResponseInputTextParam{
					Text: text,
					Type: constant.InputText("input_text"),
				},
			},
		},
		responses.EasyInputMessageRole(role),
	)
}

func defineAssistantInputParam(role, text string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemParamOfMessage(
		responses.ResponseInputMessageContentListParam{
			responses.ResponseInputContentUnionParam{
				OfInputText: &responses.ResponseInputTextParam{
					Text: text,
					Type: constant.InputText("output_text"),
				},
			},
		},
		responses.EasyInputMessageRole(role),
	)
}

func defineFileInputParam(role, filename, data string) responses.ResponseInputItemUnionParam {
	return responses.ResponseInputItemParamOfMessage(
		responses.ResponseInputMessageContentListParam{
			responses.ResponseInputContentUnionParam{
				OfInputFile: &responses.ResponseInputFileParam{
					FileData: openai.String(data),
					Filename: openai.String(filename),
					Type:     "input_file",
				},
			},
			// responses.ResponseInputContentUnionParam{
			// 	OfInputText: &responses.ResponseInputTextParam{
			// 		Text: text,
			// 		Type: "input_text",
			// 	},
			// },
		},
		responses.EasyInputMessageRole(role),
	)
}

func defineToolV3(name, description string, parameters map[string]any) responses.ToolUnionParam {
	return responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        name,
			Description: openai.String(description),
			Parameters:  openai.FunctionParameters(parameters),
		},
	}
}

func setResponseNewParams(params *responses.ResponseNewParams, args map[string]any) {
	// Whether to run the model response in the background.
	// [Learn more](https://platform.openai.com/docs/guides/background).
	if v, ok := args["background"]; ok {
		params.Background = openai.Bool(toBool(v, false))
	}

	// An upper bound for the number of tokens that can be generated for a response,
	// including visible output tokens and
	// [reasoning tokens](https://platform.openai.com/docs/guides/reasoning).
	if v, ok := args["max_output_tokens"]; ok {
		params.MaxOutputTokens = openai.Int(toInt64(v, 512))
	}

	// The maximum number of total calls to built-in tools that can be processed in a
	// response. This maximum number applies across all built-in tool calls, not per
	// individual tool. Any further attempts to call a tool by the model will be
	// ignored.
	if v, ok := args["max_tool_calls"]; ok {
		params.MaxToolCalls = openai.Int(toInt64(v, 0))
	}

	// Whether to allow the model to run tool calls in parallel.
	if v, ok := args["parallel_tool_calls"]; ok {
		params.ParallelToolCalls = openai.Bool(toBool(v, false))
	}

	// Whether to store the generated model response for later retrieval via API.
	if v, ok := args["store"]; ok {
		params.Store = openai.Bool(toBool(v, false))
	}

	// What sampling temperature to use, between 0 and 2. Higher values like 0.8 will
	// make the output more random, while lower values like 0.2 will make it more
	// focused and deterministic. We generally recommend altering this or `top_p` but
	// not both.
	if v, ok := args["temperature"]; ok {
		params.Temperature = openai.Float(toFloat64(v, 0.2))
	}

	// An integer between 0 and 20 specifying the number of most likely tokens to
	// return at each token position, each with an associated log probability.
	if v, ok := args["top_logprobs"]; ok {
		params.TopLogprobs = openai.Int(toInt64(v, 0))
	}

	// An alternative to sampling with temperature, called nucleus sampling, where the
	// model considers the results of the tokens with top_p probability mass. So 0.1
	// means only the tokens comprising the top 10% probability mass are considered.
	//
	// We generally recommend altering this or `temperature` but not both.
	if v, ok := args["top_p"]; ok {
		params.TopP = openai.Float(toFloat64(v, 0.1))
	}
}
