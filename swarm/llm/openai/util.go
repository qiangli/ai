package openai

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/middleware"
)

func NewClient(model *api.Model, vars *api.Vars) (*openai.Client, error) {
	client := openai.NewClient(
		option.WithAPIKey(model.ApiKey),
		option.WithBaseURL(model.BaseUrl),
		option.WithMiddleware(middleware.Middleware(model, vars)),
	)
	return &client, nil
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

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

func GetStrArg(key string, args map[string]any, val string) string {
	if args != nil {
		if arg, found := args[key]; found {
			if v, ok := arg.(string); ok {
				return v
			}
		}
	}
	return val
}

func GetIntArg(key string, args map[string]any, val int) int {
	if args != nil {
		if arg, found := args[key]; found {
			if str, ok := arg.(string); ok {
				if v, err := strconv.Atoi(str); err == nil {
					return v
				} else {
					return val
				}
			}
			if v, ok := arg.(int); ok {
				return v
			}
		}
	}
	return val
}
