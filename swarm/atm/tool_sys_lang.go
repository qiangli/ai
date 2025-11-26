package atm

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/lang"
)

func (r *SystemKit) ExecScript(ctx context.Context, vars *api.Vars, env *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	if tf.Body == nil {
		return nil, fmt.Errorf("missing function body: %s", tf.ID())
	}
	language := strings.ToLower(tf.Body.Language)
	code := tf.Body.Code
	if IsTemplate(code) {
		v, err := ApplyTemplate(env.Agent.Template, code, args)
		if err != nil {
			return nil, err
		}
		code = v
	}

	switch language {
	case "go", "golang":
		return lang.Go(ctx, env.FS, nil, code, nil)
	case "js", "javascript", "ecmascript":
		return lang.Javascript(ctx, code)
	case "template", "text/x-go-template":
		return code, nil
	}
	return nil, fmt.Errorf("language not supported: %s", language)
}
