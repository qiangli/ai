package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
