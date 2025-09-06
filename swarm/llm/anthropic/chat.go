package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/middleware"
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

func NewClient(model *llm.Model, vars *api.Vars) anthropic.Client {
	client := anthropic.NewClient(
		option.WithAPIKey(model.ApiKey),
		option.WithBaseURL(model.BaseUrl),
		option.WithMiddleware(middleware.Middleware(model, vars)),
	)
	return client
}

func Send(ctx context.Context, req *llm.LLMRequest) (*llm.LLMResponse, error) {
	log.Debugf(">>>ANTHROPIC:\n req: %+v\n\n", req)

	resp, err := call(ctx, req)

	log.Debugf("<<<ANTHROPIC:\n resp: %+v err: %v\n\n", resp, err)
	return resp, err
}

func call(ctx context.Context, req *llm.LLMRequest) (*llm.LLMResponse, error) {
	client := NewClient(req.Model, req.Vars)
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
			log.Errorf("role not supported: %s", v.Role)
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

	maxTurns := req.MaxTurns
	if maxTurns == 0 {
		maxTurns = 1
	}
	resp := &llm.LLMResponse{}

	// TOOD
	var temperature = anthropic.Float(0.0)

	for tries := range maxTurns {
		log.Infof("\033[33mⒶ\033[0m @%s [%v] %s %s\n", req.Agent, tries, model, req.Model.BaseUrl)

		log.Debugf("📡 *** sending request to %s ***: %v of %v\n%+v\n\n", req.Model.BaseUrl, tries, maxTurns, req)

		completion, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:       model,
			System:      system,
			Messages:    messages,
			Tools:       tools,
			MaxTokens:   8192,
			Temperature: temperature,
		})

		if err != nil {
			log.Errorf("\033[31m✗\033[0m %s\n", err)
			return nil, err
		}

		log.Infof("(%v)\n", completion.StopReason)

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
				log.Debugf("\n\n>>> tool call: %v %s props: %+v\n", i, name, props)

				//
				out, err := req.RunTool(ctx, name, props)
				var isErr bool
				if err != nil {
					out = &llm.Result{
						Value: fmt.Sprintf("%s", err),
					}
					isErr = true
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
			resp.Role = string(completion.Role)
			resp.Content = b.String()
			break
		}

		messages = append(messages, completion.ToParam())
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
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
