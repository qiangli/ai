package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// https://github.com/anthropics/anthropic-sdk-go
func defineTool(name, description string, parameters map[string]any) (*anthropic.ToolParam, error) {
	var schema = anthropic.ToolInputSchemaParam{}
	if len(parameters) > 0 {
		params := parameters["properties"]
		props := make(map[string]any)
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &props); err != nil {
			return nil, err
		}
		schema.Properties = props
	}

	if schema.Properties == nil {
		schema.Properties = map[string]interface{}{}
	}

	return &anthropic.ToolParam{
		Name:        name,
		Description: anthropic.String(description),
		InputSchema: schema,
		Type:        anthropic.ToolTypeCustom,
	}, nil
}

func NewClient(model *api.Model, apiKey string) anthropic.Client {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(model.BaseUrl),
		// option.WithMiddleware(middleware.Middleware(model, vars)),
	)
	return client
}

func Send(ctx context.Context, req *api.Request) (*api.Response, error) {
	log.GetLogger(ctx).Debugf(">ANTHROPIC:\n req: %+v\n", req)

	resp, err := call(ctx, req)

	log.GetLogger(ctx).Debugf(">ANTHROPIC:\n resp: %+v err: %v\n", resp, err)
	return resp, err
}

func call(ctx context.Context, req *api.Request) (*api.Response, error) {
	client := NewClient(req.Model, req.Token())
	model := anthropic.Model(req.Model.Model)

	var system []anthropic.TextBlockParam
	var messages []anthropic.MessageParam
	for _, v := range req.Messages {
		switch v.Role {
		case "system":
			system = append(system, anthropic.TextBlockParam{Text: v.Content})
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(v.Content)))
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(v.Content)))
		default:
			log.GetLogger(ctx).Errorf("Role not supported: %s", v.Role)
		}
	}

	var toolParams []*anthropic.ToolParam
	if len(req.Tools) > 0 {
		for _, f := range req.Tools {
			param, err := defineTool(f.ID(), f.Description, f.Parameters)
			if err != nil {
				return nil, err
			}
			toolParams = append(toolParams, param)
		}
	}
	tools := make([]anthropic.ToolUnionParam, len(toolParams))
	for i, toolParam := range toolParams {
		tools[i] = anthropic.ToolUnionParam{OfTool: toolParam}
	}

	maxTurns := req.MaxTurns()
	if maxTurns <= 0 {
		maxTurns = api.DefaultMaxTurns
	}
	resp := &api.Response{}

	// TOOD
	// https://platform.claude.com/docs/en/api/kotlin/completions/create
	var temperature = anthropic.Float(1.0)

	for tries := range maxTurns {
		log.GetLogger(ctx).Infof(api.FormatRequestLine(req, "chat", maxTurns, tries))

		log.GetLogger(ctx).Debugf("📡 sending request to %s: %v of %v\n%+v\n", req.Model.BaseUrl, tries, maxTurns, req)

		completion, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:       model,
			System:      system,
			Messages:    messages,
			Tools:       tools,
			MaxTokens:   8192,
			Temperature: temperature,
		})

		if err != nil {
			log.GetLogger(ctx).Errorf("❌ %s\n", err)
			return nil, err
		}

		log.GetLogger(ctx).Infof("(%v)\n", completion.StopReason)

		var b bytes.Buffer
		var toolResults []anthropic.ContentBlockParamUnion

		for i, block := range completion.Content {
			switch variant := block.AsAny().(type) {
			case anthropic.TextBlock:
				b.WriteString(block.Text)
				continue
			case anthropic.ToolUseBlock:
				var name = block.Name
				var props map[string]interface{}
				if err := json.Unmarshal([]byte(variant.JSON.Input.Raw()), &props); err != nil {
					return nil, err
				}
				log.GetLogger(ctx).Debugf("\n* tool call: %v %s props: %+v\n", i, name, props)

				//
				data, err := req.Runner.Run(ctx, name, props)
				var isErr bool
				var out *api.Result
				if err != nil {
					out = &api.Result{
						Value: fmt.Sprintf("%s", err),
					}
					isErr = true
				} else {
					out = api.ToResult(data)
				}

				log.GetLogger(ctx).Debugf("* tool call: %s out: %s\n", name, out)
				resp.Result = out

				// if out.State == api.StateExit {
				// 	return resp, nil
				// }
				if out.State == api.StateTransfer {
					resp.Result = out
					return resp, nil
				}

				if out.MimeType == "" || strings.HasPrefix(out.MimeType, "text/") {
					if out.MimeType == "" {
						out.MimeType = "text/plain"
					}
					toolResults = append(toolResults, newToolResultBlock(block.ID, out.Value, out.MimeType, isErr))
				} else if strings.HasPrefix(out.MimeType, "image/") {
					toolResults = append(toolResults, newToolResultBlock(block.ID, out.Value, out.MimeType, isErr))
				} else {
					toolResults = append(toolResults, newToolResultBlock(block.ID, fmt.Sprintf("mimetype not supported: %s", out.MimeType), "text/plain", true))
				}

			default:
				continue
			}
		}

		if len(toolResults) == 0 {
			resp.Result = &api.Result{
				Role:         string(completion.Role),
				MimeType:     "text/plain",
				Value:        b.String(),
				Usage:        completion.Usage,
				InputTokens:  completion.Usage.InputTokens,
				OutputTokens: completion.Usage.OutputTokens,
				//
				TotalTokens: completion.Usage.InputTokens + completion.Usage.OutputTokens,
			}
			break
		}

		messages = append(messages, completion.ToParam())
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	// not finished due the max turns reached
	if resp.Result == nil {
		return nil, fmt.Errorf("Empty response. Max turns reached: %v. Try again with a higher value for the 'max_turns' parameter", maxTurns)
	}
	return resp, nil
}

func newToolResultBlock(toolUseID string, content, mimeType string, isError bool) anthropic.ContentBlockParamUnion {
	toolBlock := anthropic.ToolResultBlockParam{
		ToolUseID: toolUseID,
		IsError:   anthropic.Bool(isError),
	}
	switch {
	case strings.HasPrefix(mimeType, "text/"):
		toolBlock.Content = []anthropic.ToolResultBlockParamContentUnion{
			{
				OfText: &anthropic.TextBlockParam{
					Text: content,
				},
			},
		}
	case strings.HasPrefix(mimeType, "image/"):
		toolBlock.Content = []anthropic.ToolResultBlockParamContentUnion{
			{
				OfImage: &anthropic.ImageBlockParam{
					Source: anthropic.ImageBlockParamSourceUnion{
						OfBase64: &anthropic.Base64ImageSourceParam{
							Data:      content,
							MediaType: anthropic.Base64ImageSourceMediaType(mimeType),
						},
					},
				},
			},
		}
	}
	return anthropic.ContentBlockParamUnion{OfToolResult: &toolBlock}
}
