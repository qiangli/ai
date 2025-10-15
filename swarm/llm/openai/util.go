package openai

import (
	"context"
	"encoding/json"
	"sync"

	// "github.com/openai/openai-go/v2"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// run all the tool calls and return the results in the same sequence as the tool call list
// func runTools(
// 	parent context.Context,
// 	runner api.ToolRunner,
// 	toolCalls []openai.ChatCompletionMessageToolCallUnion,
// 	max int,
// ) []*api.Result {
// 	calls := make([]*ToolCall, len(toolCalls))
// 	for i, v := range toolCalls {
// 		calls[i] = &ToolCall{
// 			ID:        v.ID,
// 			Name:      v.Function.Name,
// 			Arguments: v.Function.Arguments,
// 		}
// 	}
// 	return runToolsV3(parent, runner, calls, max)
// }

func runToolsV3(
	parent context.Context,
	runner api.ToolRunner,
	calls []*ToolCall,
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
	runner api.ToolRunner,
	toolCalls []*ToolCall,
	max int,
) []*api.Result {
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, max)
	results := make([]*api.Result, len(toolCalls))

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	log.GetLogger(parent).Debugf("\n* tool call count: %v", len(toolCalls))

	for i, toolCall := range toolCalls {
		wg.Add(1)

		go func(i int, toolCall *ToolCall) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			var name = toolCall.Name
			var props map[string]any
			if err := json.Unmarshal([]byte(toolCall.Arguments), &props); err != nil {
				results[i] = &api.Result{
					Value: err.Error(),
				}
				return
			}

			log.GetLogger(parent).Debugf("\n* tool call: %v %s props: %+v\n", i, name, props)

			out, err := runner(ctx, name, props)
			if err != nil {
				results[i] = &api.Result{
					Value: err.Error(),
				}
				return
			}

			log.GetLogger(parent).Debugf("\n* tool call: %s out: %s\n", name, out)
			results[i] = out

			// 	if out.State == api.StateExit || out.State == api.StateTransfer {
			if out.State == api.StateExit {
				cancel()
			}
		}(i, toolCall)
	}

	wg.Wait()

	close(semaphore)

	return results
}

func runTool(
	ctx context.Context,
	runner api.ToolRunner,
	toolCall *ToolCall,
) *api.Result {
	var name = toolCall.Name
	var props map[string]any
	if err := json.Unmarshal([]byte(toolCall.Arguments), &props); err != nil {
		return &api.Result{
			Value: err.Error(),
		}
	}

	log.GetLogger(ctx).Debugf("\n* tool call: %s props: %+v\n", name, props)

	out, err := runner(ctx, name, props)
	if err != nil {
		return &api.Result{
			Value: err.Error(),
		}
	}

	log.GetLogger(ctx).Debugf("\n* tool call: %s out: %s\n", name, out)
	return out
}
