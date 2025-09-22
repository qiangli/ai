package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
)

// https://ai.google.dev/gemini-api/docs/models

// https://ai.google.dev/gemini-api/docs/function-calling?example=meeting
// https://github.com/google-gemini/api-examples/blob/main/go/function_calling.go
func defineTool(name, description string, parameters map[string]any) (*genai.FunctionDeclaration, error) {
	var schema *genai.Schema

	if len(parameters) > 0 {
		var desc string
		var props map[string]*genai.Schema
		var required []string

		if obj, found := parameters["description"]; found {
			if s, ok := obj.(string); ok {
				desc = s
			}
		}
		if obj, found := parameters["properties"]; found {
			props = make(map[string]*genai.Schema)
			data, err := json.Marshal(obj)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, &props); err != nil {
				return nil, err
			}
		}
		if obj, found := parameters["required"]; found {
			if a, ok := obj.([]string); ok {
				required = a
			}
		}
		schema = &genai.Schema{
			Type:        genai.TypeObject,
			Description: desc,
			Properties:  props,
			Required:    required,
		}
	}

	return &genai.FunctionDeclaration{
		Name:        name,
		Description: description,
		Parameters:  schema,
	}, nil
}

func NewClient(ctx context.Context, apiKey, _ string) (*genai.Client, error) {
	// GOOGLE_GEMINI_BASE_URL
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
		// TODO middleware?
	})
	return client, err
}

func Send(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debug(">>>GEMINI:\n req: %+v\n\n", req)

	resp, err := call(ctx, req)

	log.GetLogger(ctx).Debug("<<<GEMINI:\n resp: %+v err: %v\n\n", resp, err)
	return resp, err
}

func call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	client, err := NewClient(
		ctx,
		req.Model.ApiKey,
		req.Model.BaseUrl,
	)
	if err != nil {
		return nil, err
	}

	var messages []*genai.Content

	for _, v := range req.Messages {
		switch v.Role {
		case "system", "assistant":
			messages = append(messages, genai.NewContentFromText(v.Content, genai.RoleModel))
		case "user":
			messages = append(messages, genai.NewContentFromText(v.Content, genai.RoleUser))
		default:
			// just ignore and move on
			log.GetLogger(ctx).Error("role not supported: %s", v.Role)
		}
	}

	var config *genai.GenerateContentConfig

	if len(req.Tools) > 0 {
		// kits := make(map[string][]*llm.ToolFunc)
		// for _, f := range req.Tools {
		// 	fa, ok := kits[f.Kit]
		// 	if !ok {
		// 		fa = []*llm.ToolFunc{}
		// 	}
		// 	fa = append(fa, f)
		// 	kits[f.Kit] = fa
		// }

		// var tools []*genai.Tool
		// for _, kit := range kits {
		// 	var fds []*genai.FunctionDeclaration
		// 	for _, f := range kit {
		// 		fd, err := defineTool(f.ID, f.Description, f.Parameters)
		// 		if err != nil {
		// 			return nil, err
		// 		}
		// 		fds = append(fds, fd)
		// 	}
		// 	tools = append(tools, &genai.Tool{
		// 		FunctionDeclarations: fds,
		// 	})
		// }

		// TODO verify - the toolkit structure is flattened
		var tools []*genai.Tool
		for _, f := range req.Tools {
			var fds []*genai.FunctionDeclaration
			fd, err := defineTool(f.ID(), f.Description, f.Parameters)
			if err != nil {
				return nil, err
			}
			fds = append(fds, fd)
			tools = append(tools, &genai.Tool{
				FunctionDeclarations: fds,
			})
		}

		// Set up the generate content configuration with function calling enabled.
		config = &genai.GenerateContentConfig{
			Tools: tools,
			ToolConfig: &genai.ToolConfig{
				FunctionCallingConfig: &genai.FunctionCallingConfig{
					// The mode equivalent to FunctionCallingConfigMode.ANY in JS.
					Mode: genai.FunctionCallingConfigModeAny,
					// Mode: genai.FunctionCallingConfigModeAuto,
				},
			},
		}
	}

	maxTurns := req.MaxTurns
	if maxTurns == 0 {
		maxTurns = 1
	}
	resp := &llm.Response{}

	model := req.Model.Model

	for tries := range maxTurns {
		log.GetLogger(ctx).Info("\033[33mⒼ\033[0m @%s [%v] %s %s\n", req.Agent, tries, model, req.Model.BaseUrl)

		log.GetLogger(ctx).Debug("📡 *** sending request to %s ***: %v of %v\n%+v\n\n", req.Model.BaseUrl, tries, maxTurns, req)

		completion, err := client.Models.GenerateContent(ctx, model, messages, config)
		if err != nil {
			log.GetLogger(ctx).Error("\033[31m✗\033[0m %s\n", err)
			return nil, err
		}
		log.GetLogger(ctx).Info("(%v)\n", "done")

		// https://ai.google.dev/gemini-api/docs/function-calling?example=meeting
		toolCalls := completion.FunctionCalls()
		if len(toolCalls) == 0 {
			resp.Role = ""
			resp.Content = completion.Text()
			break
		}

		// call tools
		for i, toolCall := range toolCalls {
			// var id = toolCall.ID
			var name = toolCall.Name
			var args = toolCall.Args

			log.GetLogger(ctx).Debug("\n\n>>> tool call: %v %s args: %+v\n", i, name, args)

			//
			out, err := req.RunTool(ctx, name, args)
			if err != nil {
				out = &api.Result{
					Value: fmt.Sprintf("%s", err),
				}
			}

			log.GetLogger(ctx).Debug("\n<<< tool call: %s out: %+v\n", name, out)
			resp.Result = out

			if out.State == api.StateExit {
				resp.Content = out.Value
				return resp, nil
			}
			if out.State == api.StateTransfer {
				return resp, nil
			}

			// Gemini seems to require the exact pairing of the call and result messages
			// call message
			// messages = []*genai.Content{}
			messages = append(messages, &genai.Content{
				Parts: []*genai.Part{
					{
						FunctionCall: toolCall,
					},
				},
				Role: genai.RoleModel,
			})

			// result message
			if out.MimeType != "" {
				messages = append(messages, genai.NewContentFromParts(
					[]*genai.Part{
						{
							InlineData: &genai.Blob{
								Data:     []byte(out.Value),
								MIMEType: out.MimeType,
							},
						},
					}, genai.RoleUser))
			} else {
				messages = append(messages, genai.NewContentFromText(out.Value, genai.RoleUser))
				// messages = append(messages, &genai.Content{
				// 	Parts: []*genai.Part{
				// 		{
				// 			FunctionResponse: &genai.FunctionResponse{
				// 				ID:   id,
				// 				Name: name,
				// 				Response: map[string]any{
				// 					"output": out,
				// 					"error":  err,
				// 				},
				// 			},
				// 		},
				// 	},
				// 	Role: "function",
				// })
			}
		}
	}

	return resp, nil
}
