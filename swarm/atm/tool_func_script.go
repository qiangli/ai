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
	mime := strings.ToLower(tf.Body.MimeType)
	code := tf.Body.Script
	if api.IsTemplate(code) {
		v, err := CheckApplyTemplate(env.Agent.Template, code, EncodeArgs(args))
		if err != nil {
			return nil, err
		}
		code = v
	}

	switch mime {
	case "application/x-sh", "bash", "sh":
		return vars.RootAgent.Shell.Run(ctx, code, args)
	case "application/yaml", "yaml", "yml":
		return nil, fmt.Errorf("mime type not supported: %s", mime)
	case "application/x-go", "go", "golang":
		return lang.Golang(ctx, vars, nil, code, nil)
	case "text/javascript", "js", "javascript", "ecmascript":
		return lang.Javascript(ctx, code)
	case "text/x-go-template", "template", "tpl":
		return code, nil
	}
	return nil, fmt.Errorf("mime type not supported: %s", mime)
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
