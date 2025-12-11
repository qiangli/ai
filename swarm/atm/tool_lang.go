package atm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/lang"
)

// Execute script in go, js, and go-template
func (r *FuncKit) ExecScript(ctx context.Context, vars *api.Vars, env *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	if tf.Body == nil {
		return nil, fmt.Errorf("missing function body: %s", tf.ID())
	}
	language := strings.ToLower(tf.Body.Language)
	code := tf.Body.Code
	if IsTemplate(code) {
		v, err := ApplyTemplate(env.Agent.Template, code, EncodeArgs(args))
		if err != nil {
			return nil, err
		}
		code = v
	}

	switch language {
	case "go", "golang":
		return lang.Golang(ctx, vars, nil, code, nil)
	case "js", "javascript", "ecmascript":
		return lang.Javascript(ctx, code)
	case "template", "text/x-go-template":
		return code, nil
	}
	return nil, fmt.Errorf("language not supported: %s", language)
}

// return a copy of the origin map but encoded in json string for array and object
func EncodeArgs(v map[string]any) map[string]any {
	var args = make(map[string]any)
	for k, v := range v {
		switch v.(type) {
		case bool, int, int8, int16, int32, int64, float32, float64, string:
			args[k] = v
		default:
			jsonValue, err := json.Marshal(v)
			if err != nil {
				args[k] = v
				continue
			}
			args[k] = string(jsonValue)
		}
	}
	return args
}
