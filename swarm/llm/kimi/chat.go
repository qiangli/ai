package kimi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

const maxThreadLimit = 3

// NewClient creates an OpenAI-style client configured for Kimi's base URL and API key.
func NewClient(model *api.Model, apiKey string) (*openai.Client, error) {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(model.BaseUrl),
	)
	return &client, nil
}

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

func runTools(
	parent context.Context,
	runner api.ActionRunner,
	calls []*api.ToolCall,
	max int,
) []*api.Result {
	switch len(calls) {
	case 0:
		return nil
	case 1:
		return []*api.Result{runTool(parent, runner, calls[0])}
	default:
		return runToolsInParallel(parent, runner, calls, max)
	}
}

func runToolsInParallel(
	parent context.Context,
	runner api.ActionRunner,
	toolCalls []*api.ToolCall,
	max int,
) []*api.Result {
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, max)
	results := make([]*api.Result, len(toolCalls))

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	for i, toolCall := range toolCalls {
		wg.Add(1)
		go func(i int, toolCall *api.ToolCall) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			var name = toolCall.Command
			var args = toolCall.Arguments
			var props = make(map[string]any)
			if args != nil {
				args.Copy(props)
			}

			data, err := runner.Run(parent, name, props)
			if err != nil {
				results[i] = &api.Result{Value: err.Error()}
				return
			}

			out := api.ToResult(data)
			results[i] = out
		}(i, toolCall)
	}

	wg.Wait()
	close(semaphore)
	return results
}

func runTool(
	ctx context.Context,
	runner api.ActionRunner,
	toolCall *api.ToolCall,
) *api.Result {
	var name = toolCall.Command
	var args = toolCall.Arguments
	var props = make(map[string]any)
	if args != nil {
		args.Copy(props)
	}

	data, err := runner.Run(ctx, name, props)
	if err != nil {
		return &api.Result{Value: err.Error()}
	}
	out := api.ToResult(data)
	return out
}

// Send sends the request to Kimi using an OpenAI-compatible client.
// This mirrors the xai adapter approach and is text-only.
func Send(ctx context.Context, req *api.Request) (*api.Response, error) {
	log.GetLogger(ctx).Debugf(">KIMI: req: %+v\n", req)

	// derive model name for Kimi: prefer req.Model.Model but fall back and sanitize
	var kimiModel string
	if req.Model != nil {
		kimiModel = req.Model.Model
		if kimiModel == "" {
			kimiModel = req.Model.String()
		}
		if strings.Contains(kimiModel, "/") {
			parts := strings.Split(kimiModel, "/")
			kimiModel = parts[len(parts)-1]
		}
	}
	if kimiModel == "" {
		kimiModel = "kimi-k2-fast"
		log.GetLogger(ctx).Infof("WARN: KIMI: model not specified, defaulting to %q\n", kimiModel)
	}

	if req.Model == nil {
		return nil, fmt.Errorf("no model provided")
	}

	client, err := NewClient(req.Model, req.Token())
	if err != nil {
		log.GetLogger(ctx).Errorf("❌ %s\n", err)
		return nil, err
	}

	var params = openai.ChatCompletionNewParams{
		Model: kimiModel,
	}

	if req.Arguments != nil {
		// reuse xai's param mapping style if needed; for now no-op
	}

	if len(req.Messages) > 0 {
		var messages []openai.ChatCompletionMessageParamUnion
		for _, v := range req.Messages {
			switch v.Role {
			case "system":
				messages = append(messages, openai.SystemMessage(v.Content))
			case "assistant":
				messages = append(messages, openai.AssistantMessage(v.Content))
			case "user":
				messages = append(messages, openai.UserMessage(v.Content))
			default:
				// unsupported role
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

	maxTurns := req.MaxTurns()
	if maxTurns <= 0 {
		maxTurns = api.DefaultMaxTurns
	}

	resp := &api.Response{}

	for tries := range maxTurns {
		log.GetLogger(ctx).Infof(api.FormatRequestLine(req, "chat", maxTurns, tries))
		log.GetLogger(ctx).Debugf("📡 sending chat request to %s: %v of %v\n%+v\n", req.Model.BaseUrl, tries, maxTurns, req)

		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			log.GetLogger(ctx).Errorf("❌ %s\n", err)
			return nil, err
		}

		log.GetLogger(ctx).Infof("(%s)\n", completion.Choices[0].FinishReason)

		toolCalls := completion.Choices[0].Message.ToolCalls
		if len(toolCalls) == 0 {
			resp.Result = &api.Result{
				Role:     string(completion.Choices[0].Message.Role),
				MimeType: "text/plain",
				Value:    completion.Choices[0].Message.Content,
				Usage:    completion.Usage,
			}
			// map tokens (CompletionUsage is a value type in the openai client)
			resp.Result.InputTokens = completion.Usage.PromptTokens
			resp.Result.OutputTokens = completion.Usage.CompletionTokens
			resp.Result.TotalTokens = completion.Usage.TotalTokens
			return resp, nil
		}

		// append assistant message
		params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())

		calls := make([]*api.ToolCall, len(toolCalls))
		for i, v := range toolCalls {
			var props map[string]any
			if err := json.Unmarshal([]byte(v.Function.Arguments), &props); err != nil {
				return nil, err
			}
			calls[i] = api.NewToolCall(v.ID, v.Function.Name, props)
		}

		results := runTools(ctx, req.Runner, calls, maxThreadLimit)
		for i, out := range results {
			if out == nil {
				params.Messages = append(params.Messages, openai.ToolMessage("no result", calls[i].CallID))
				continue
			}
			if out.State == api.StateTransfer || out.State == api.StateExit {
				resp.Result = out
				return resp, nil
			}
			params.Messages = append(params.Messages, openai.ToolMessage(out.Value, calls[i].CallID))
		}
	}

	if resp.Result == nil {
		return nil, fmt.Errorf("Empty response. Max turns reached: %v. Try again with a higher value for the 'max_turns' parameter", maxTurns)
	}
	return resp, nil
}
