package openai

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared/constant"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/middleware"
)

// https://platform.openai.com/docs/api-reference/responses
// https://platform.openai.com/docs/guides/function-calling
// https://github.com/csotherden/openai-go-responses-examples/tree/main
func NewClientV3(model *api.Model, vars *api.Vars) (*openai.Client, error) {
	client := openai.NewClient(
		option.WithAPIKey(model.ApiKey),
		option.WithBaseURL(model.BaseUrl),
		option.WithMiddleware(middleware.Middleware(model, vars)),
	)
	return &client, nil
}

func SendV3(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI v3:\n req: %+v\n", req)

	var err error
	var resp *llm.Response

	resp, err = respond(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI v3:\n resp: %+v err: %v\n", resp, err)
	return resp, err
}

func respond(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	client, err := NewClientV3(req.Model, req.Vars)
	if err != nil {
		return nil, err
	}

	var resp = &llm.Response{}

	params := responses.ResponseNewParams{
		Model: req.Model.Model,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(req.Message),
		},
		// Temperature:     openai.Float(0.7),
		// MaxOutputTokens: openai.Int(512),
	}

	if len(req.Messages) > 0 {
		var messages []responses.ResponseInputItemUnionParam
		for _, v := range req.Messages {
			switch v.Role {
			// case "system", "developer":
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

	//
	result, err := client.Responses.New(ctx, params)
	if err != nil {
		return resp, err
	}

	params.PreviousResponseID = openai.String(result.ID)
	params.Input = responses.ResponseNewParamsInputUnion{}

	var calls []*ToolCall
	for _, output := range result.Output {
		if output.Type == "function_call" {
			v := output.AsFunctionCall()
			calls = append(calls, &ToolCall{
				ID:        v.CallID,
				Name:      v.Name,
				Arguments: v.Arguments,
			})
		}
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

	// No tools calls made, we already have our final response
	if len(params.Input.OfInputItemList) == 0 {
		resp.Result = &api.Result{
			Value: result.OutputText(),
		}
		return resp, nil
	}

	// Make a final call with our tools results and no tools to get the final output
	params.Tools = nil
	result, err = client.Responses.New(ctx, params)
	if err != nil {
		return resp, err
	}

	resp.Result = &api.Result{
		Value: result.OutputText(),
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
