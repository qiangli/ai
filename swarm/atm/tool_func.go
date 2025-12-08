package atm

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/tool/memory"
)

type FuncKit struct {
	kgm *memory.KGManager
}

func NewFuncKit() *FuncKit {
	return &FuncKit{
		kgm: memory.NewKGManager(),
	}
}

func (r *FuncKit) Call(ctx context.Context, vars *api.Vars, env *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	// if tf.Body == nil || (tf.Body.Url == "" && tf.Body.Code == "") {
	// 	return nil, fmt.Errorf("no function body: %s", tf.ID())
	// }

	if tf.Body == nil {
		return r.builtin(ctx, vars, env, tf, args)
	}

	language := strings.ToLower(tf.Body.Language)
	switch language {
	case "go", "golang", "js", "javascript", "ecmascript", "template", "text/x-go-template":
		return r.ExecScript(ctx, vars, env, tf, args)
	case "py", "python":
		return r.DO(ctx, vars, env, tf, args)
	}

	return nil, fmt.Errorf("language not supported: %s", language)
}

func (r *FuncKit) builtin(ctx context.Context, vars *api.Vars, _ *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return CallKit(r, tf.Kit, tf.Name, callArgs...)
}
