package openai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

const maxThreadLimit = 3

func NewClient(model *api.Model, token string) (*openai.Client, error) {
	client := openai.NewClient(
		option.WithAPIKey(token),
		option.WithBaseURL(model.BaseUrl),
		// option.WithMiddleware(middleware.Middleware(model, vars)),
	)
	return &client, nil
}

// https://platform.openai.com/docs/api-reference/responses
// https://platform.openai.com/docs/guides/function-calling
// https://github.com/csotherden/openai-go-responses-examples/tree/main
func NewClientV3(model *api.Model, token string) (*openai.Client, error) {
	client := openai.NewClient(
		option.WithAPIKey(token),
		option.WithBaseURL(model.BaseUrl),
		// option.WithMiddleware(middleware.Middleware(model, vars)),
	)
	return &client, nil
}

// type ToolCall struct {
// 	ID        string
// 	Name      string
// 	Arguments string
// }

func runToolsV3(
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

	log.GetLogger(parent).Debugf("\n* tool call count: %v", len(toolCalls))

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

			var name = toolCall.Name
			// var props = toolCall.Arguments
			// if err := json.Unmarshal([]byte(toolCall.Arguments), &props); err != nil {
			// 	results[i] = &api.Result{
			// 		Value: err.Error(),
			// 	}
			// 	return
			// }
			var args = toolCall.Arguments
			var props = make(map[string]any)
			if args != nil {
				args.Copy(props)
			}
			log.GetLogger(parent).Debugf("\n* tool call: %v %s props: %+v\n", i, name, props)

			data, err := runner.Run(ctx, name, props)
			if err != nil {
				results[i] = &api.Result{
					Value: err.Error(),
				}
				return
			}

			out := api.ToResult(data)

			log.GetLogger(parent).Debugf("\n* tool call: %s out: %s\n", name, out)

			results[i] = out

			// if out.State == api.StateExit {
			// 	cancel()
			// }
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
	var name = toolCall.Name
	var args = toolCall.Arguments
	var props = make(map[string]any)
	if args != nil {
		args.Copy(props)
	}
	// var props map[string]any
	// if err := json.Unmarshal([]byte(toolCall.Arguments), &props); err != nil {
	// 	return &api.Result{
	// 		Value: err.Error(),
	// 	}
	// }

	log.GetLogger(ctx).Debugf("\n* tool call: %s props: %+v\n", name, props)

	data, err := runner.Run(ctx, name, props)
	if err != nil {
		return &api.Result{
			Value: err.Error(),
		}
	}
	out := api.ToResult(data)
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

func toBool(x any, val bool) bool {
	if v, ok := x.(bool); ok {
		return v
	}
	if v, err := strconv.ParseBool(fmt.Sprintf("%s", x)); err == nil {
		return v
	}
	return val
}

func toInt64(x any, val int64) int64 {
	if v, ok := x.(int64); ok {
		return v
	}
	if v, err := strconv.ParseInt(fmt.Sprintf("%s", x), 0, 64); err == nil {
		return v
	}
	return val
}

func toFloat64(x any, val float64) float64 {
	if v, ok := x.(float64); ok {
		return v
	}
	if v, err := strconv.ParseFloat(fmt.Sprintf("%s", x), 64); err == nil {
		return v
	}
	return val
}

// resource supports the following:
// + data:<mime>;base64,
// + http:// and https://
// + file:/
// assume local file if no scheme is provided
func fetchContent(resource string) (io.Reader, error) {
	// Check for data URI
	if strings.HasPrefix(resource, "data:") {
		commaIndex := strings.Index(resource, ",")
		if commaIndex == -1 {
			return nil, errors.New("invalid data URL")
		}
		data := resource[commaIndex+1:]
		reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(data))
		return reader, nil
	}

	// Check for HTTP/HTTPS URL
	if strings.HasPrefix(resource, "http://") || strings.HasPrefix(resource, "https://") {
		resp, err := http.Get(resource)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}

	// file URI or local file
	// Assume local file if no scheme is provided
	p := strings.TrimPrefix(resource, "file:/")
	p = filepath.Join("/", p)
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func formatReason(s string) string {
	// The reason the model stopped generating tokens. This will be `stop` if the model
	// hit a natural stop point or a provided stop sequence, `length` if the maximum
	// number of tokens specified in the request was reached, `content_filter` if
	// content was omitted due to a flag from our content filters, `tool_calls` if the
	// model called a tool, or `function_call` (deprecated) if the model called a
	// function.
	//
	// Any of "stop", "length", "tool_calls", "content_filter", "function_call".
	switch s {
	case "stop":
		return "stop"
	case "length":
		return "maximum tokens reached"
	case "tool_calls":
		return "tool called"
	case "content_filter":
		return "content filtered"
	case "function_call":
		return "function called"
	default:
		return s
	}
}
